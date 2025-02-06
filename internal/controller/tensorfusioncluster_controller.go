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
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	utils "github.com/NexusGPU/tensor-fusion-operator/internal/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/equality"
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
// +kubebuilder:rbac:groups=apps,resources=deployments;namespaces;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
func (r *TensorFusionClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling TensorFusionCluster", "name", req.NamespacedName.Name)
	defer func() {
		log.Info("Finished reconciling TensorFusionCluster", "name", req.NamespacedName.Name)
	}()

	// Get the TensorFusionConnection object
	tfc := &tfv1.TensorFusionCluster{}
	if err := r.Get(ctx, req.NamespacedName, tfc); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request, return without error
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get TensorFusionCluster")
		return ctrl.Result{}, err
	}

	deleted, err := utils.HandleFinalizer(ctx, tfc, r.Client, func(context context.Context, tfc *tfv1.TensorFusionCluster) error {
		log.Info("TensorFusionCluster is being deleted", "name", tfc.Name)
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	if deleted {
		return ctrl.Result{}, nil
	}

	if tfc.Status.Phase == "" {
		tfc.SetAsPending()
		if err := r.mustUpdateTFClusterStatus(ctx, tfc); err != nil {
			return ctrl.Result{}, err
		}
		// Next loop to make sure the custom resources are created
		return ctrl.Result{Requeue: true}, nil
	}

	// when running or pending, make sure the custom resources are aligned, if changed, set as updating
	if tfc.Status.Phase == constants.PhasePending || tfc.Status.Phase == constants.PhaseRunning {
		if changed, err := r.mustReconcileGPUPool(ctx, tfc); err != nil {
			if changed {
				tfc.SetAsUpdating()
			}
			return ctrl.Result{}, err
		}

		if changed, err := r.mustReconcileTimeSeriesDatabase(ctx, tfc); err != nil {
			if changed {
				tfc.SetAsUpdating()
			}
			return ctrl.Result{}, err
		}

		if changed, err := r.mustReconcileCloudVendorConnection(ctx, tfc); err != nil {
			if changed {
				tfc.SetAsUpdating()
			}
			return ctrl.Result{}, err
		}

		if tfc.Status.Phase == constants.PhaseUpdating {
			if err := r.mustUpdateTFClusterStatus(ctx, tfc); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// when updating, check util they are ready
	if tfc.Status.Phase == constants.PhaseUpdating {
		// check status, if not ready, requeue after backoff delay, if all components are ready, set as ready
		if ready, conditions, err := r.checkTFClusterComponentsReady(ctx, tfc); err != nil {
			return ctrl.Result{}, err
		} else if !ready {
			// update retry count
			tfc.Status.RetryCount = tfc.Status.RetryCount + 1
			// store additional check result for each component
			tfc.SetAsUpdating(conditions...)

			if err := r.mustUpdateTFClusterStatus(ctx, tfc); err != nil {
				return ctrl.Result{}, err
			}
			delay := utils.CalculateExponentialBackoffWithJitter(tfc.Status.RetryCount)
			return ctrl.Result{RequeueAfter: delay}, nil
		} else {
			// all components are ready, set cluster as ready
			tfc.SetAsReady(conditions...)
			if err := r.mustUpdateTFClusterStatus(ctx, tfc); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

	}

	tfc.SetAsUnknown(fmt.Errorf("reconcile logic not correct, reached unknown phase, cluster: %s", tfc.Name))
	if err := r.mustUpdateTFClusterStatus(ctx, tfc); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TensorFusionClusterReconciler) mustReconcileTimeSeriesDatabase(ctx context.Context, tfc *tfv1.TensorFusionCluster) (bool, error) {
	// TODO: Not implemented yet
	return false, nil
}

func (r *TensorFusionClusterReconciler) mustReconcileCloudVendorConnection(ctx context.Context, tfc *tfv1.TensorFusionCluster) (bool, error) {
	// TODO: Not implemented yet
	return false, nil
}

func (r *TensorFusionClusterReconciler) mustReconcileGPUPool(ctx context.Context, tfc *tfv1.TensorFusionCluster) (bool, error) {
	// Fetch existing GPUPools that belong to this cluster
	var gpupoolsList tfv1.GPUPoolList
	err := r.List(ctx, &gpupoolsList, client.MatchingLabels(map[string]string{
		constants.LabelKeyOwner: tfc.GetName(),
	}))
	if err != nil {
		return false, fmt.Errorf("failed to list GPUPools: %w", err)
	}

	// Map existing GPUPools by their unique identifier (e.g., name)
	existingGPUPools := make(map[string]*tfv1.GPUPool)
	for _, pool := range gpupoolsList.Items {
		if pool.OwnerReferences != nil {
			owner := &metav1.OwnerReference{}
			for i := range pool.OwnerReferences {
				if controllerRef := pool.OwnerReferences[i]; controllerRef.Name == tfc.GetName() {
					owner = &controllerRef
					break
				}
			}
			if owner != nil && owner.Name == tfc.GetName() {
				existingGPUPools[pool.Name] = &pool
			}
		}
	}

	errors := []error{}
	anyPoolChanged := false

	// Process each intended GPUPool in the cluster spec
	for _, poolSpec := range tfc.Spec.GPUPools {
		key := tfc.Name + "-" + poolSpec.Name

		// Check if the GPUPool already exists
		existingPool := existingGPUPools[key]
		if existingPool == nil {
			poolLabels := map[string]string{
				constants.LabelKeyOwner: tfc.GetName(),
			}
			for k, v := range tfc.Labels {
				poolLabels[k] = v
			}

			// Create new GPUPool
			gpupool := &tfv1.GPUPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   key,
					Labels: poolLabels,
				},
				Spec: poolSpec.SpecTemplate,
			}
			e := controllerutil.SetControllerReference(tfc, gpupool, scheme.Scheme)
			if e != nil {
				errors = append(errors, fmt.Errorf("failed to set controller reference: %w", e))
				continue
			}
			err = r.Create(ctx, gpupool)
			anyPoolChanged = true
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to create GPUPool %s: %w", key, err))
				continue
			}
		} else {
			// Update existing GPUPool if spec changed
			if !equality.Semantic.DeepEqual(&existingPool.Spec, &poolSpec.SpecTemplate) {
				existingPool.Spec = poolSpec.SpecTemplate
				err = r.Update(ctx, existingPool)
				if err != nil {
					errors = append(errors, fmt.Errorf("failed to update GPUPool %s: %w", key, err))
				}
				anyPoolChanged = true
			}
		}
	}

	// Delete any GPUPools that are no longer in the spec
	for poolName := range existingGPUPools {
		found := false
		for _, poolSpec := range tfc.Spec.GPUPools {
			key := tfc.Name + "-" + poolSpec.Name
			if key == poolName {
				found = true
				break
			}
		}
		if !found {
			existingPool := existingGPUPools[poolName]
			err = r.Delete(ctx, existingPool)
			anyPoolChanged = true
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to delete GPUPool %s: %w", poolName, err))
			}
		}
	}

	// If there were any errors, return them; otherwise, set status to Running
	if len(errors) > 0 {
		for _, err := range errors {
			return false, fmt.Errorf("reconcile pool in tensor-fusion cluster failed: %w", err)
		}
	}
	return anyPoolChanged, nil
}

func (r *TensorFusionClusterReconciler) checkTFClusterComponentsReady(ctx context.Context, tfc *tfv1.TensorFusionCluster) (bool, []metav1.Condition, error) {
	allPass := true
	conditions := []metav1.Condition{
		{
			Type:   constants.ConditionStatusTypeGPUPool,
			Status: metav1.ConditionTrue,
		},
		{
			Type:   constants.ConditionStatusTypeTimeSeriesDatabase,
			Status: metav1.ConditionTrue,
		},
		{
			Type:   constants.ConditionStatusTypeCloudVendorConnection,
			Status: metav1.ConditionTrue,
		},
	}

	// check if all conditions are true, any not ready component will make allPass false

	// Step 1. check GPUPools
	var pools tfv1.GPUPoolList
	err := r.List(ctx, &pools, client.MatchingLabels(map[string]string{
		constants.LabelKeyOwner: tfc.GetName(),
	}))
	if err != nil {
		return false, nil, fmt.Errorf("failed to list GPUPools: %w", err)
	}
	if len(pools.Items) != len(tfc.Spec.GPUPools) {
		allPass = false
		conditions[0].Status = metav1.ConditionFalse
	} else {
		for i := range pools.Items {
			if pools.Items[i].Status.Phase != constants.PhaseRunning {
				allPass = false
				conditions[0].Status = metav1.ConditionFalse
				break
			}
		}
	}

	// Step 2. check TimeSeriesDatabase, TODO

	// Step 3. check CloudVendorConnection, TODO

	return allPass, conditions, nil
}

func (r *TensorFusionClusterReconciler) mustUpdateTFClusterStatus(ctx context.Context, tfc *tfv1.TensorFusionCluster) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// Get the latest version of the tfc
		latestTFCluster := &tfv1.TensorFusionCluster{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      tfc.Name,
			Namespace: tfc.Namespace,
		}, latestTFCluster); err != nil {
			return err
		}

		// Update the status fields we care about
		latestTFCluster.Status = tfc.Status

		// Update the cluster status
		if err := r.Status().Update(ctx, latestTFCluster); err != nil {
			return err
		}
		return nil
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *TensorFusionClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.TensorFusionCluster{}).
		Named("tensorfusioncluster").
		Owns(&tfv1.GPUPool{}).
		Complete(r)
}
