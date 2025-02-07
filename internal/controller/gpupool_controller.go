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

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	utils "github.com/NexusGPU/tensor-fusion-operator/internal/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/finalizers,verbs=update
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

	// sync the GPU Pool into memory, used by scheduler and mutation webhook
	r.GpuPoolState.Set(pool.Name, &pool.Spec)

	// TODO, any GPUNode changes trigger GPUPool reconcile, it should change the status, aggregate the total amount of resources, update current status
	if err := r.startNodeDiscoverys(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}
	// TODO, when componentConfig changed, it should notify corresponding resource to upgrade
	return ctrl.Result{}, nil
}

func (r *GPUPoolReconciler) startNodeDiscoverys(
	ctx context.Context,
	pool *tfv1.GPUPool,
) error {
	log := log.FromContext(ctx)
	log.Info("Starting node node discovery job")

	// TODO: need to write a interval in go coroutine to check if node could be compacted like Karpenter, when it's ok to mark as destroying, change the status and trigger a reconcile
	// if it's AutoSelect mode, stop all Pods on it, and let ClusterAutoscaler or Karpenter to delete the node
	// if it's Provision mode, stop all Pods on it, and destroy the Node from cloud provider

	// Strategy #1: check if any empty node can be deleted

	// Strategy #2: check if whole Pool can be bin-packing into less nodes, check from low-priority to high-priority nodes one by one, if workloads could be moved to other nodes (using a simulated scheduler), evict it and mark cordoning, let scheduler to re-schedule

	// Strategy #3: check if any node can be reduced to 1/2 size. for remaining nodes, check if allocated size < 1/2 * total size, if so, check if can buy smaller instance

	podTmpl := &corev1.PodTemplate{}
	err := json.Unmarshal(pool.Spec.ComponentConfig.NodeDiscovery.PodTemplate.Raw, podTmpl)
	if err != nil {
		return fmt.Errorf("unmarshal pod template: %w", err)
	}
	// pool.Spec.NodeManagerConfig.NodeSelector
	nodes := &corev1.NodeList{}
	if err := r.Client.List(ctx, nodes); err != nil {
		return fmt.Errorf("list nodes: %v", err)
	}

	for _, node := range nodes.Items {
		matches, err := schedulingcorev1.MatchNodeSelectorTerms(&node, pool.Spec.NodeManagerConfig.NodeSelector)
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
					Name: fmt.Sprintf("node-discovery-%s", node.Name),
					// TODO: 	How to get the current ns
					Namespace: "",
				},
				Spec: batchv1.JobSpec{
					TTLSecondsAfterFinished: ptr.To[int32](3600 * 10),
					Template:                *templateCopy,
				},
			}
			if err := r.Get(ctx, client.ObjectKeyFromObject(job), job); err != nil {
				if err := ctrl.SetControllerReference(pool, job, r.Scheme); err != nil {
					return fmt.Errorf("set owner reference %w", err)
				}
				if err := r.Create(ctx, job); err != nil {
					return fmt.Errorf("create node discovery job %w", err)
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
			// TODO: this watch should with predicate to avoid performance impact
			node := obj.(*tfv1.GPUNode)
			if poolName, exists := node.Annotations[constants.PoolIdentifierAnnotationKey]; exists {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: poolName}}}
			}
			return nil
		})).
		Complete(r)
}
