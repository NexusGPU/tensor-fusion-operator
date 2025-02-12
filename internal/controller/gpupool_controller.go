/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	utils "github.com/NexusGPU/tensor-fusion-operator/internal/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	schedulingcorev1 "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GPUPoolReconciler reconciles a GPUPool object
type GPUPoolReconciler struct {
	client.Client
	GpuPoolState config.GpuPoolState
	GpuNodeState config.GpuNodeState
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/finalizers,verbs=update

// Reconcile GPU pools
func (r *GPUPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling GPUPool", "name", req.NamespacedName.Name)
	defer func() {
		log.Info("Finished reconciling GPUPool", "name", req.NamespacedName.Name)
	}()

	pool := &tfv1.GPUPool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		if errors.IsNotFound(err) {
			r.GpuPoolState.Delete(pool.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	deleted, err := utils.HandleFinalizer(ctx, pool, r.Client, func(ctx context.Context, pool *tfv1.GPUPool) error {
		r.GpuPoolState.Delete(pool.Name)
		// TODO: stop all existing workers and hypervisors, stop time series flow aggregations
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	if deleted {
		return ctrl.Result{}, nil
	}

	// For provisioning mode, check if need to scale up GPUNodes upon AvailableCapacity changed
	isProvisioningMode := pool.Spec.NodeManagerConfig != nil &&
		pool.Spec.NodeManagerConfig.NodeProvisioner != nil &&
		pool.Spec.NodeManagerConfig.NodeProvisioner.Mode == tfv1.NodeProvisionerModeNative

	// sync the GPU Pool into memory, used by scheduler and mutation webhook
	r.GpuPoolState.Set(pool.Name, &pool.Spec)

	// Provisioning mode, check capacity and scale up if needed
	if isProvisioningMode {
		newNodeCreated, err := r.reconcilePoolCapacityWithProvisioner(ctx, pool)
		if err != nil {
			return ctrl.Result{}, err
		}
		// Set phase to updating and let GPUNode event trigger the check and update capacity loop, util all nodes are ready
		if newNodeCreated {
			pool.Status.Phase = tfv1.TensorFusionPoolPhaseUpdating
			err = r.Client.Status().Update(ctx, pool)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	// TODO, any GPUNode changes trigger GPUPool reconcile, it should change the status, aggregate the total amount of resources, update current status
	// THIS NEED TO MOVE INTO GPU NODE CONTROLLER, rather than POOL CONTROLLER
	if !isProvisioningMode {
		if err := r.startNodeDiscovery(ctx, pool); err != nil {
			return ctrl.Result{}, err
		}
	}
	// TODO, when componentConfig changed, it should notify corresponding resource to upgrade
	// eg. when hypervisor changed, should change all owned GPUNode's status.phase to Updating

	if err := r.reconcilePoolCurrentCapacityAndReadiness(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GPUPoolReconciler) reconcilePoolCurrentCapacityAndReadiness(ctx context.Context, pool *tfv1.GPUPool) error {
	log := log.FromContext(ctx)

	nodes := &tfv1.GPUNodeList{}
	if err := r.Client.List(ctx, nodes); err != nil {
		return fmt.Errorf("list nodes of Pool %s failed: %v", pool.Name, err)
	}

	log.Info("Calculate current capacity and readiness for pool", "name", pool.Name)

	totalGPUs := int32(0)
	readyNodes := 0
	totalVRAM := resource.Quantity{}
	virtualVRAM := resource.Quantity{}
	totalTFlops := resource.Quantity{}
	virtualTFlops := resource.Quantity{}

	for _, node := range nodes.Items {
		totalGPUs = totalGPUs + node.Status.TotalGPUs
		totalVRAM.Add(node.Status.TotalVRAM)
		totalTFlops.Add(node.Status.TotalTFlops)
		if node.Status.Phase == tfv1.TensorFusionGPUNodePhaseRunning {
			readyNodes++
		}
		virtualVRAM.Add(node.Status.VirtualVRAM)
		virtualTFlops.Add(node.Status.VirtualTFlops)
	}

	pool.Status.TotalGPUs = totalGPUs
	pool.Status.TotalNodes = int32(len(nodes.Items))
	pool.Status.TotalVRAM = totalVRAM
	pool.Status.TotalTFlops = totalTFlops

	pool.Status.ReadyNodes = int32(readyNodes)
	pool.Status.NotReadyNodes = int32(len(nodes.Items)) - pool.Status.ReadyNodes

	pool.Status.VirtualTFlops = virtualTFlops
	pool.Status.VirtualVRAM = virtualVRAM

	if readyNodes == len(nodes.Items) {
		pool.Status.Phase = tfv1.TensorFusionPoolPhaseRunning
		log.Info("Pool is running, all nodes are ready", "name", pool.Name, "nodes", len(nodes.Items))
	} else {
		// set back to updating, wait GPUNode change triggering the pool change
		pool.Status.Phase = tfv1.TensorFusionPoolPhasePending
	}

	if err := r.Client.Status().Update(ctx, pool); err != nil {
		return fmt.Errorf("update pool status: %w", err)
	}
	return nil
}

func (r *GPUPoolReconciler) startNodeDiscovery(
	ctx context.Context,
	pool *tfv1.GPUPool,
) error {
	log := log.FromContext(ctx)
	log.Info("Starting node discovery job")

	if pool.Spec.NodeManagerConfig == nil || pool.Spec.NodeManagerConfig.NodeSelector == nil {
		log.Info("missing NodeManagerConfig.nodeSelector config in pool spec, skipped")
		return nil
	}
	if pool.Spec.ComponentConfig == nil || pool.Spec.ComponentConfig.NodeDiscovery.PodTemplate == nil {
		return fmt.Errorf(`missing node discovery pod template in pool spec`)
	}
	podTmpl := &corev1.PodTemplate{}
	err := json.Unmarshal(pool.Spec.ComponentConfig.NodeDiscovery.PodTemplate.Raw, podTmpl)
	if err != nil {
		return fmt.Errorf("unmarshal pod template: %w", err)
	}
	selector := labels.NewSelector()
	poolReq, err := labels.NewRequirement(fmt.Sprintf(constants.GPUNodePoolIdentifierLabelFormat, pool.Name), selection.DoubleEquals, []string{"true"})
	if err != nil {
		return fmt.Errorf("new GPUNodePoolIdentifier label seletor: %w", err)
	}
	selector = selector.Add(*poolReq)
	nodes := &tfv1.GPUNodeList{}
	if err := r.Client.List(ctx, nodes, &client.ListOptions{LabelSelector: selector}); err != nil {
		return fmt.Errorf("list gpunodes: %v", err)
	}

	for _, gpuNode := range nodes.Items {
		node := &corev1.Node{}
		if err := r.Client.Get(ctx, client.ObjectKeyFromObject(&gpuNode), node); err != nil {
			return err
		}
		matches, err := schedulingcorev1.MatchNodeSelectorTerms(node, pool.Spec.NodeManagerConfig.NodeSelector)
		if err != nil {
			return err
		}
		if matches {
			templateCopy := podTmpl.Template.DeepCopy()
			if templateCopy.Spec.Affinity == nil {
				templateCopy.Spec.Affinity = &corev1.Affinity{}
			}
			if templateCopy.Spec.Affinity.NodeAffinity == nil {
				templateCopy.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
			}
			if templateCopy.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
				templateCopy.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
					NodeSelectorTerms: make([]corev1.NodeSelectorTerm, 0),
				}
			}
			templateCopy.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms =
				append(templateCopy.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, corev1.NodeSelectorTerm{
					MatchFields: []corev1.NodeSelectorRequirement{
						{
							Key:      "metadata.name",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{node.Name},
						},
					},
				})
			// create node-discovery job
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("node-discovery-%s", node.Name),
					Namespace: utils.CurrentNamespace(),
				},
				Spec: batchv1.JobSpec{
					TTLSecondsAfterFinished: ptr.To[int32](3600 * 10),
					Template:                *templateCopy,
				},
			}
			if err := r.Get(ctx, client.ObjectKeyFromObject(job), job); err != nil {
				if errors.IsNotFound(err) {
					if err := ctrl.SetControllerReference(pool, job, r.Scheme); err != nil {
						return fmt.Errorf("set owner reference %w", err)
					}
					if err := r.Create(ctx, job); err != nil {
						return fmt.Errorf("create node discovery job %w", err)
					}
				} else {
					return fmt.Errorf("create node job %w", err)
				}
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GPUPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.GPUPool{}).
		Named("gpupool").
		Watches(&tfv1.GPUNode{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			requests := []reconcile.Request{}

			node := obj.(*tfv1.GPUNode)
			for labelKey := range node.Labels {
				if strings.HasPrefix(labelKey, constants.GPUNodePoolIdentifierLabelPrefix) {
					tmp := strings.Split(labelKey, "/")
					if len(tmp) != 3 {
						continue
					}
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{Name: tmp[2]},
					})
				}
			}
			return requests
		})).
		Complete(r)
}
