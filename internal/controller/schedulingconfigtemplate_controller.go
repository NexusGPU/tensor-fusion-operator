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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tensorfusionaiv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
)

// SchedulingConfigTemplateReconciler reconciles a SchedulingConfigTemplate object
type SchedulingConfigTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=schedulingconfigtemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=schedulingconfigtemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=schedulingconfigtemplates/finalizers,verbs=update

// When deleted, need check if any GPU pool is using this template, if so, add warning event and requeue
// When updated, trigger the GPU pool status update for scheduling part
func (r *SchedulingConfigTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulingConfigTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tensorfusionaiv1.SchedulingConfigTemplate{}).
		Named("schedulingconfigtemplate").
		Complete(r)
}
