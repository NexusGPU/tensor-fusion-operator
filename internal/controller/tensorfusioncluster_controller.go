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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tensorfusionaiv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
)

var (
	tensorFusionClusterFinalizer = constants.TensorFusionFinalizer
)

// TensorFusionClusterReconciler reconciles a TensorFusionCluster object
type TensorFusionClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=tensorfusionclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=tensorfusionclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=tensorfusionclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Cluster reconcile logic:
//  1. When created
//     1.1. Enroll to cloud
//     1.2. Add finalizer
//     1.3. Create or Update the GPUPool CR
//     1.4. Create or connect to PG Database if configured, if PG settings changed, update PG CR
//     1.5. Start data pipelines CronJob to aggregate data and report to cloud, if pipeline settings changed, restart the pipelines
//     1.6. Start Cron schedules if current instance is master
//     1.7. Create or update the cloud vendor connection if configured
//     1.8. Send anonymous telemetry data to cloud if turn on
//  2. If DeletionTimestamp is less than 0
//     2.1. Check if all workloads are deleted, if not, throw an warning event and requeue
//     2.2. Delete GPUPool
//     2.3. Delete PG Database if provisioned by TensorFusion
//     2.4. UnEnroll from cloud
//     2.6. Stop data pipelines CronJob
//     2.5. Remove finalizer and let Kubernetes delete the object
//  4. Cron Schedule
//     4.1 Update the GPU cluster capacity by aggregating all pools resources
//     4.2 Heartbeat with cloud, check license status, if expired, update status and requeue, after 30 days, stop reporting to cloud and remove data pipelines, stop scheduler, after 60 days, stop all existing workers and hypervisors
//     4.3 Heartbeat with storage vendor, check PG database status
func (r *TensorFusionClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	tfc := &tensorfusionaiv1.TensorFusionCluster{}
	err := r.Get(ctx, req.NamespacedName, tfc)
	if err != nil {
		log.FromContext(ctx).Error(err, "unable to fetch TensorFusionCluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// add a finalizer to the object
	if !containsString(tfc.Finalizers, tensorFusionClusterFinalizer) {
		tfc.Finalizers = append(tfc.Finalizers, tensorFusionClusterFinalizer)
		err = r.Update(ctx, tfc)
		if err != nil {
			log.FromContext(ctx).Error(err, "unable to update TensorFusionCluster")
			return ctrl.Result{}, err
		}
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if tfc.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then we should add the finalizer and update the object. Finally we
		// return and requeue the object so that we can pick it up again after
		// updating it.
		if !containsString(tfc.Finalizers, tensorFusionClusterFinalizer) {
			tfc.Finalizers = append(tfc.Finalizers, tensorFusionClusterFinalizer)
			if err := r.Update(ctx, tfc); err != nil {
				log.FromContext(ctx).Error(err, "unable to update TensorFusionCluster")
				return ctrl.Result{}, err
			}
			// we return and requeue the object so that we can pick it up again after updating it
			return ctrl.Result{}, nil
		}
	} else {
		// The object is being deleted
		if containsString(tfc.Finalizers, tensorFusionClusterFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.Delete(ctx, tfc); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			tfc.Finalizers = removeString(tfc.Finalizers, tensorFusionClusterFinalizer)
			if err := r.Update(ctx, tfc); err != nil {
				log.FromContext(ctx).Error(err, "unable to remove finalizer from TensorFusionCluster")
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TensorFusionClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tensorfusionaiv1.TensorFusionCluster{}).
		Named("tensorfusioncluster").
		Complete(r)
}
