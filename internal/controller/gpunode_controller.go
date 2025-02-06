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
	"os"

	"github.com/NexusGPU/tensor-fusion-operator/internal/config"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
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
		Complete(r)
}

func (r *GPUNodeReconciler) reconcileHypervisorPod(ctx context.Context, node *tfv1.GPUNode) error {
	poolName := node.Annotations[constants.PoolIdentifierAnnotationKey]
	if poolName == "" {
		log.FromContext(ctx).Info("Orphan GPU node found, not belongs to any pool, skipping reconcile.", "node", node.Name)
		return nil
	}

	namespace := constants.NamespaceDefaultVal
	if os.Getenv(constants.NamespaceEnv) != "" {
		namespace = os.Getenv(constants.NamespaceEnv)
	}
	hypervisorPodName := fmt.Sprintf("%s-%s-hypervisor", poolName, node.Name)
	pool := r.GpuPoolState.Get(poolName)
	if pool == nil {
		return fmt.Errorf("failed to get tensor-fusion pool, can not create hypervisor pod, pool: %s", poolName)
	}
	hypervisorConfig := pool.ComponentConfig.Hypervisor

	hypervisorPod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: hypervisorPodName, Namespace: namespace}, hypervisorPod)

	if errors.IsNotFound(err) {
		podTmpl := &corev1.PodTemplate{}
		err := json.Unmarshal(hypervisorConfig.PodTemplate.Raw, podTmpl)
		if err != nil {
			return fmt.Errorf("failed to unmarshal pod template: %w", err)
		}
		spec := podTmpl.Template.Spec
		if spec.NodeSelector == nil {
			spec.NodeSelector = make(map[string]string)
		}
		spec.NodeSelector["kubernetes.io/hostname"] = node.Name

		newPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hypervisorPodName,
				Namespace: namespace,
				Labels: map[string]string{
					constants.PoolIdentifierAnnotationKey: poolName,
				},
			},
			Spec: spec,
		}

		e := controllerutil.SetControllerReference(node, newPod, scheme.Scheme)
		if e != nil {
			return fmt.Errorf("failed to set controller reference: %w", e)
		}

		err = r.Create(ctx, newPod)
		if err != nil {
			return fmt.Errorf("failed to create hypervisorPod: %v", err)
		}
		return nil
	}
	// TODO: handle update and new generation case, new Pool config case etc.

	return nil
}
