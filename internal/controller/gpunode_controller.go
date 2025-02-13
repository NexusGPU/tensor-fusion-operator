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

	cloudprovider "github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider"
	"github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/types"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"github.com/NexusGPU/tensor-fusion-operator/internal/utils"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GPUNodeReconciler reconciles a GPUNode object
type GPUNodeReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	GpuPoolState config.GpuPoolState
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpunodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpunodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpunodes/finalizers,verbs=update

// Reconcile GPU nodes
func (r *GPUNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling GPUNode", "name", req.NamespacedName.Name)
	defer func() {
		log.Info("Finished reconciling GPUNode", "name", req.NamespacedName.Name)
	}()

	node := &tfv1.GPUNode{}
	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	deleted, err := utils.HandleFinalizer(ctx, node, r.Client, func(ctx context.Context, node *tfv1.GPUNode) error {
		if node.Spec.ManageMode == tfv1.GPUNodeManageModeAutoSelect {
			// Do nothing, but if it's managed by Karpenter, should come up with some way to tell Karpenter to terminate the GPU node
		} else if node.Spec.ManageMode == tfv1.GPUNodeManageModeProvisioned {
			clusterName := node.GetLabels()[constants.LabelKeyClusterOwner]
			cluster := &tfv1.TensorFusionCluster{}
			if err := r.Get(ctx, client.ObjectKey{Name: clusterName}, cluster); err != nil {
				return err
			}

			vendorCfg := cluster.Spec.ComputingVendor
			if vendorCfg == nil {
				return fmt.Errorf("failed to get computing vendor config for cluster %s", clusterName)
			}

			provider, err := cloudprovider.GetProvider(*vendorCfg)
			if err != nil {
				return err
			}
			(*provider).TerminateNode(ctx, &types.NodeIdentityParam{
				InstanceID: node.Status.NodeInfo.InstanceID,
				Region:     node.Status.NodeInfo.Region,
			})

		}
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	if deleted {
		return ctrl.Result{}, nil
	}

	// Node discovery should be here

	if err := r.reconcileHypervisorPod(ctx, node); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GPUNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.GPUNode{}).
		Named("gpunode").
		Owns(&corev1.Node{}).
		Complete(r)
}

func (r *GPUNodeReconciler) reconcileHypervisorPod(ctx context.Context, node *tfv1.GPUNode) error {
	namespace := utils.CurrentNamespace()
	log := log.FromContext(ctx)
	for labelKey := range node.Labels {
		if strings.HasPrefix(labelKey, constants.GPUNodePoolIdentifierLabelPrefix) {
			segment := strings.Split(labelKey, "/")
			if len(segment) != 3 {
				log.Info("Invalid label key for GPU node", "key", labelKey, "node", node.Name)
				continue
			}
			poolName := segment[2]

			hypervisorPodName := fmt.Sprintf("%s-%s-hypervisor", poolName, node.Name)
			pool := r.GpuPoolState.Get(poolName)
			if pool == nil {
				return fmt.Errorf("failed to get tensor-fusion pool, can not create hypervisor pod, pool: %s", poolName)
			}
			hypervisorConfig := pool.ComponentConfig.Hypervisor
			podTmpl := &corev1.PodTemplate{}
			err := json.Unmarshal(hypervisorConfig.PodTemplate.Raw, podTmpl)
			if err != nil {
				return fmt.Errorf("failed to unmarshal pod template: %w", err)
			}
			spec := podTmpl.Template.Spec.DeepCopy()
			if spec.NodeSelector == nil {
				spec.NodeSelector = make(map[string]string)
			}
			spec.NodeSelector["kubernetes.io/hostname"] = node.Name
			newPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hypervisorPodName,
					Namespace: namespace,
					Labels: map[string]string{
						fmt.Sprintf(constants.GPUNodePoolIdentifierLabelFormat, poolName): "true",
					},
				},
				Spec: *spec,
			}

			e := controllerutil.SetControllerReference(node, newPod, r.Scheme)
			if e != nil {
				return fmt.Errorf("failed to set controller reference: %w", e)
			}
			_, err = controllerutil.CreateOrUpdate(ctx, r.Client, newPod, func() error {
				log.Info("Creating or Updating hypervisor pod", "name", hypervisorPodName)
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to create or update hypervisor pod: %w", err)
			}
		}
	}
	return nil
}

// TODO after node info updated, trigger a reconcile loop to complement virtual capacity and other status from operator side
func (r *GPUNodeReconciler) CalculateVirtualCapacity(node *tfv1.GPUNode, pool *tfv1.GPUPool) (resource.Quantity, resource.Quantity) {
	diskSize, _ := node.Status.NodeInfo.DataDiskSize.AsInt64()
	ramSize, _ := node.Status.NodeInfo.RAMSize.AsInt64()

	virtualVRAM := node.Status.TotalVRAM.DeepCopy()
	virtualTFlops := node.Status.TotalTFlops.DeepCopy()

	virtualVRAM.Add(*resource.NewQuantity(
		int64(float64(float64(diskSize)*float64(pool.Spec.CapacityConfig.Oversubscription.VRAMExpandToHostDisk)/100.0)),
		resource.DecimalSI),
	)
	virtualVRAM.Add(*resource.NewQuantity(
		int64(float64(float64(ramSize)*float64(pool.Spec.CapacityConfig.Oversubscription.VRAMExpandToHostMem)/100.0)),
		resource.DecimalSI),
	)
	virtualTFlops.Mul(int64(pool.Spec.CapacityConfig.Oversubscription.TFlopsOversellRatio))
	return virtualVRAM, virtualTFlops
}
