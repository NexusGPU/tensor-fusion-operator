package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"time"

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

func CalculateExponentialBackoffWithJitter(retryCount int64) time.Duration {
	const (
		baseDelay  = 3 * time.Second
		maxDelay   = 60 * time.Second
		factor     = 1.5
		maxRetries = 10
	)

	if retryCount > maxRetries {
		retryCount = maxRetries
	}

	backoff := float64(baseDelay) * math.Pow(factor, float64(retryCount))

	jitter := rand.Float64() * backoff

	totalDelay := time.Duration(jitter)
	if totalDelay < baseDelay {
		totalDelay = baseDelay
	}
	if totalDelay > maxDelay {
		totalDelay = maxDelay
	}

	return totalDelay
}

func CurrentNamespace() string {
	namespace := constants.NamespaceDefaultVal
	envNamespace := os.Getenv(constants.NamespaceEnv)
	if envNamespace != "" {
		namespace = envNamespace
	}
	return namespace
}

func GetObjectHash(obj any) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v", obj)))
	return hex.EncodeToString(hasher.Sum(nil))
}
