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

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GPUPoolReconciler reconciles a GPUPool object
type GPUPoolReconciler struct {
	client.Client
	GpuPoolState config.GpuPoolState
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/finalizers,verbs=update
func (r *GPUPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pool := &tfv1.GPUPool{}
	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		if errors.IsNotFound(err) {
			r.GpuPoolState.Delete(pool.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	r.GpuPoolState.Set(pool.Name, &pool.Spec)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GPUPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.GPUPool{}).
		Named("gpupool").
		Complete(r)
}
