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

	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"github.com/NexusGPU/tensor-fusion-operator/internal/utils"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// GPUNodeReconciler reconciles a GPUNode object
type GPUNodeReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	GpuPoolState config.GpuPoolState
	GpuNodeState config.GpuNodeState
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
		WithEventFilter(
			predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					r.GpuNodeState.Set(e.Object.(*tfv1.GPUNode).Name, e.Object.(*tfv1.GPUNode))
					return true
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					r.GpuNodeState.Set(e.ObjectNew.(*tfv1.GPUNode).Name, e.ObjectNew.(*tfv1.GPUNode))
					return true
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					r.GpuNodeState.Delete(e.Object.(*tfv1.GPUNode).Name)
					return true
				},
			},
		).
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
						fmt.Sprintf(constants.GPUNodePoolIdentifierLabelKey, poolName): "true",
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
