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

var (
	tensorFusionClusterFinalizer = constants.Finalizer
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
		// TODO: stop all existing workers and hypervisors, stop time series flow aggregations
		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	if deleted {
		return ctrl.Result{}, nil
	}

	if err := r.mustReconcileGPUPool(ctx, tfc); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.mustReconcileTimeSeriesDatabase(ctx, tfc); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.mustReconcileCloudVendorConnection(ctx, tfc); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.mustUpdateTFClusterStatus(ctx, tfc); err != nil {
		return ctrl.Result{}, err
	}

	tfc.SetAsReady()

	return ctrl.Result{}, nil
}

func (r *TensorFusionClusterReconciler) mustReconcileTimeSeriesDatabase(ctx context.Context, tfc *tfv1.TensorFusionCluster) error {
	// TODO: Not implemented yet
	return nil
}

func (r *TensorFusionClusterReconciler) mustReconcileCloudVendorConnection(ctx context.Context, tfc *tfv1.TensorFusionCluster) error {
	// TODO: Not implemented yet
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TensorFusionClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.TensorFusionCluster{}).
		Named("tensorfusioncluster").
		Owns(&tfv1.GPUPool{}).
		Complete(r)
}

func (r *TensorFusionClusterReconciler) mustReconcileGPUPool(ctx context.Context, tfc *tfv1.TensorFusionCluster) error {
	// Fetch existing GPUPools that belong to this cluster
	var gpupoolsList tfv1.GPUPoolList
	err := r.List(ctx, &gpupoolsList, client.MatchingFields{".metadata.controller": tfc.Name})
	if err != nil {
		return fmt.Errorf("failed to list GPUPools: %w", err)
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

	// Process each intended GPUPool in the cluster spec
	for _, poolSpec := range tfc.Spec.GPUPools {
		key := tfc.Name + "-" + poolSpec.Name

		// Check if the GPUPool already exists
		existingPool := existingGPUPools[key]
		if existingPool == nil {
			// Create new GPUPool
			gpupool := &tfv1.GPUPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:   key,
					Labels: tfc.Labels,
				},
				Spec: poolSpec.Spec,
			}
			controllerutil.SetControllerReference(tfc, gpupool, scheme.Scheme)
			err = r.Create(ctx, gpupool)
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to create GPUPool %s: %w", key, err))
				continue
			}
		} else {
			// Update existing GPUPool if spec changed
			if !equality.Semantic.DeepEqual(&existingPool.Spec, &poolSpec.Spec) {
				existingPool.Spec = poolSpec.Spec
				err = r.Update(ctx, existingPool)
				if err != nil {
					errors = append(errors, fmt.Errorf("failed to update GPUPool %s: %w", key, err))
				}
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
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to delete GPUPool %s: %w", poolName, err))
			}
		}
	}

	// If there were any errors, return them; otherwise, set status to Running
	if len(errors) > 0 {
		for _, err := range errors {
			return fmt.Errorf("reconcile failed: %w", err)
		}
	}
	return nil
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
