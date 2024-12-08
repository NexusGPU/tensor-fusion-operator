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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	scheduler "github.com/NexusGPU/tensor-fusion-operator/internal/scheduler"
	"github.com/NexusGPU/tensor-fusion-operator/internal/worker"
)

// TensorFusionConnectionReconciler reconciles a TensorFusionConnection object
type TensorFusionConnectionReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Scheduler scheduler.Scheduler
}

var (
	tensorFusionConnectionFinalizer = constants.TensorFusionFinalizer
)

// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=tensorfusionconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=tensorfusionconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tensor-fusion.ai,resources=tensorfusionconnections/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TensorFusionConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get the TensorFusionConnection object
	connection := &tfv1.TensorFusionConnection{}
	if err := r.Get(ctx, req.NamespacedName, connection); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request, return without error
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get TensorFusionConnection")
		return ctrl.Result{}, err
	}

	// Check if the connection is being deleted
	if !connection.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if containsString(connection.Finalizers, tensorFusionConnectionFinalizer) {
			// Our finalizer is present, so let's handle our external dependency
			if err := r.handleDeletion(ctx, connection); err != nil {
				return ctrl.Result{}, err
			}

			// Remove our finalizer from the list and update it
			connection.Finalizers = removeString(connection.Finalizers, tensorFusionConnectionFinalizer)
			if err := r.Update(ctx, connection); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Our finalizer has finished, so the reconciler can do nothing
		return ctrl.Result{}, nil
	}

	// Add finalizer if it's not present
	if !containsString(connection.Finalizers, tensorFusionConnectionFinalizer) {
		connection.Finalizers = append(connection.Finalizers, tensorFusionConnectionFinalizer)
		if err := r.Update(ctx, connection); err != nil {
			return ctrl.Result{}, err
		}
		// Return here as the update will trigger another reconciliation
		return ctrl.Result{}, nil
	}

	var gpu *tfv1.GPU
	// If status is not set or pending, try to schedule
	if connection.Status.Phase == "" || connection.Status.Phase == tfv1.TensorFusionConnectionPending {
		// Try to get an available gpu from scheduler
		var err error
		gpu, err = r.Scheduler.Schedule(connection.Spec.Resources.Requests)
		if err != nil {
			log.Info(err.Error())
			connection.Status.Phase = tfv1.TensorFusionConnectionPending
		} else if gpu != nil {
			connection.Status.Phase = tfv1.TensorFusionConnectionRunning
			connection.Status.ConnectionURL = worker.GenerateConnectionURL(gpu, connection)
			// Store the gpu name for cleanup
			connection.Status.GPU = gpu.Name
		} else {
			connection.Status.Phase = tfv1.TensorFusionConnectionPending
		}
	}

	if err := r.MustUpdateStatus(ctx, connection, gpu); err != nil {
		return ctrl.Result{}, err
	}

	if connection.Status.Phase == tfv1.TensorFusionConnectionPending {
		return ctrl.Result{RequeueAfter: constants.PendingRequeueDuration}, nil
	}
	return ctrl.Result{}, nil
}

// handleDeletion handles cleanup of external dependencies
func (r *TensorFusionConnectionReconciler) handleDeletion(ctx context.Context, connection *tfv1.TensorFusionConnection) error {
	if connection.Status.GPU == "" {
		return nil // No gpu was allocated, nothing to clean up
	}

	// Get the gpu
	gpu := &tfv1.GPU{}
	if err := r.Get(ctx, client.ObjectKey{Name: connection.Status.GPU}, gpu); err != nil {
		if errors.IsNotFound(err) {
			// gpu is already gone, nothing to do
			return nil
		}
		return err
	}

	// Release the resources
	if err := r.Scheduler.Release(connection.Spec.Resources.Requests, gpu); err != nil {
		return err
	}

	return r.MustUpdateStatus(ctx, connection, gpu)
}

// Helper functions to handle finalizers
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := []string{}
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func (r *TensorFusionConnectionReconciler) MustUpdateStatus(ctx context.Context, connection *tfv1.TensorFusionConnection, gpu *tfv1.GPU) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// Get the latest version of the connection
		latestConnection := &tfv1.TensorFusionConnection{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      connection.Name,
			Namespace: connection.Namespace,
		}, latestConnection); err != nil {
			return err
		}

		// Update the status fields we care about
		latestConnection.Status = connection.Status

		// Update the connection status
		if err := r.Status().Update(ctx, latestConnection); err != nil {
			return err
		}

		if gpu != nil {
			// Get the latest version of the gpu
			latestgpu := &tfv1.GPU{}

			if err := r.Get(ctx, client.ObjectKey{
				Name:      gpu.Name,
				Namespace: gpu.Namespace,
			}, latestgpu); err != nil {
				return err
			}

			// Update the status fields we care about
			latestgpu.Status.Available = gpu.Status.Available
			if err := r.Status().Update(ctx, latestgpu); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *TensorFusionConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tfv1.TensorFusionConnection{}).
		Named("tensorfusionconnection").
		Complete(r)
}
