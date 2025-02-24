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

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	"github.com/NexusGPU/tensor-fusion-operator/internal/metrics"
	webhookv1 "github.com/NexusGPU/tensor-fusion-operator/internal/webhook/v1"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Add GPU connection for Pods using GPU
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	pod := &corev1.Pod{}

	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}
	poolName, resources := webhookv1.ParseTFResources(pod)
	if len(resources) == 0 {
		return ctrl.Result{}, nil
	}

	// generate tensor fusion connections and apply to cluster
	tfConnections := generateTensorFusionConnection(pod, poolName, resources)

	for _, tfConnection := range tfConnections {
		existConn := &tfv1.TensorFusionConnection{}
		if err := r.Get(ctx, types.NamespacedName{Name: tfConnection.Name, Namespace: tfConnection.Namespace}, existConn); err != nil {
			if errors.IsNotFound(err) {
				if err := r.Client.Create(ctx, tfConnection); err != nil {
					return ctrl.Result{}, fmt.Errorf("create connection(%s) : %w", tfConnection.Namespace+"/"+tfConnection.Name, err)
				}
			}
		}
	}

	// update metrics
	for _, res := range resources {
		labels := prometheus.Labels{
			"pod":       pod.Name,
			"namespace": pod.Namespace,
			"container": res.ContainerName,
		}
		metrics.GpuTflopsRequest.With(labels).Set(res.TflopsRequest.AsApproximateFloat64())
		metrics.GpuTflopsLimit.With(labels).Set(res.TflopsLimit.AsApproximateFloat64())
		metrics.VramBytesRequest.With(labels).Set(res.VramRequest.AsApproximateFloat64())
		metrics.VramBytesLimit.With(labels).Set(res.VramLimit.AsApproximateFloat64())
	}

	return ctrl.Result{}, nil
}

func generateTensorFusionConnection(pod *corev1.Pod, poolName string, tfReq []webhookv1.TFResource) []*tfv1.TensorFusionConnection {
	connections := make([]*tfv1.TensorFusionConnection, 0, len(tfReq))

	for _, req := range tfReq {
		connection := &tfv1.TensorFusionConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.ConnectionName,
				Namespace: req.ConnectionNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "v1",
						Kind:       "Pod",
						Name:       pod.Name,
						UID:        pod.UID,
					},
				},
			},
			Spec: tfv1.TensorFusionConnectionSpec{
				PoolName: poolName,
				Resources: tfv1.Resources{
					Requests: tfv1.Resource{
						Tflops: req.TflopsRequest,
						Vram:   req.VramRequest,
					},
					Limits: tfv1.Resource{
						Tflops: req.TflopsLimit,
						Vram:   req.VramLimit,
					},
				},
			},
			Status: tfv1.TensorFusionConnectionStatus{
				Phase: tfv1.TensorFusionConnectionPending,
			},
		}
		connections = append(connections, connection)
	}

	return connections
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.Domain + "/enabled": "true",
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create predicate: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}, builder.WithPredicates(p)).
		Named("pod").
		Complete(r)
}
