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
	"errors"
	"fmt"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tensorfusionaiv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/utils"
)

// GPUPoolReconciler reconciles a GPUPool object
type GPUPoolReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=gpupools/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// 1. Start or select GPU nodes, create GPUNode CR
// 2. Initialize the resource aggregation job of this Pool if not started
// 3. Deploy hypervisor and preload images of different components, maintain component status
func (r *GPUPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	debugLogger := log.FromContext(ctx).V(3).WithValues("GPUPool", req.Name)
	logger := log.FromContext(ctx).WithValues("GPUPool", req.Name)

	debugLogger.Info("Reconciling GPUPool")
	defer func() {
		debugLogger.Info("Reconciliation loop end")
	}()

	logger.Info("GPUPool %s", req.String())

	pool, err := r.getGPUPool(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	if pool == nil {
		// TODO
		return ctrl.Result{}, nil
	}

	// Run the inner reconcile loop. Translate any ErrNextLoop to an errorless return
	result, err := r.reconcile(ctx, pool)
	if errors.Is(err, utils.ErrNextLoop) {
		return result, nil
	}
	if errors.Is(err, utils.ErrTerminateLoop) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	return result, nil
}

func (r *GPUPoolReconciler) reconcile(ctx context.Context, gpupool *tensorfusionaiv1.GPUPool) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GPUPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tensorfusionaiv1.GPUPool{}).
		Named("gpupool").
		Complete(r)
}

func (r *GPUPoolReconciler) getGPUPool(
	ctx context.Context,
	req ctrl.Request,
) (*tensorfusionaiv1.GPUPool, error) {
	contextLogger := log.FromContext(ctx)
	gpupool := &tensorfusionaiv1.GPUPool{}
	if err := r.Get(ctx, req.NamespacedName, gpupool); err != nil {
		if apierrs.IsNotFound(err) {
			contextLogger.Info("Resource has been deleted")
			return nil, nil
		}
		return nil, fmt.Errorf("cannot get the managed resource: %w", err)
	}

	return gpupool, nil
}
