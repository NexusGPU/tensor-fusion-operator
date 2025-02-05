package utils

import (
	"context"
	"errors"

	constants "github.com/NexusGPU/tensor-fusion-operator/internal/constants"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ErrNextLoop is not a real error. It forces the current reconciliation loop to stop
// and return the associated Result object
var ErrNextLoop = errors.New("stop this loop and return the associated Result object")

// ErrTerminateLoop is not a real error. It forces the current reconciliation loop to stop
var ErrTerminateLoop = errors.New("stop this loop and do not requeue")

func HandleFinalizer[T client.Object](ctx context.Context, obj T, r client.Client, deleteHook func(context.Context, T) error) (bool, error) {
	// Check if object is being deleted
	deleted := !obj.GetDeletionTimestamp().IsZero()
	if deleted {
		// Object is being deleted - process finalizer
		if controllerutil.ContainsFinalizer(obj, constants.Finalizer) {
			// Run custom deletion hook
			if err := deleteHook(ctx, obj); err != nil {
				return false, err
			}

			// Remove finalizer once cleanup is done
			controllerutil.RemoveFinalizer(obj, constants.Finalizer)
			if err := r.Update(ctx, obj); err != nil {
				return false, err
			}
		}
	} else {
		// Object is not being deleted - add finalizer if not present
		if !controllerutil.ContainsFinalizer(obj, constants.Finalizer) {
			controllerutil.AddFinalizer(obj, constants.Finalizer)
			if err := r.Update(ctx, obj); err != nil {
				return false, err
			}
		}
	}
	return deleted, nil
}
