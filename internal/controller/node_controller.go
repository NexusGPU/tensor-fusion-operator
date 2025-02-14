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
	"os"
	"strings"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PodReconciler reconciles a Pod object
type NodeReconciler struct {
	client.Client
	PoolState config.GpuPoolState
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes/finalizers,verbs=update

// This reconcile loop only take effect on nodeSelector mode, while in AutoProvision mode, GPUNode will manage the K8S Node rather than reversed
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	node := &corev1.Node{}
	if err := r.Get(ctx, req.NamespacedName, node); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Node")
		return ctrl.Result{}, err
	}

	// Remove deletion mark if updated
	if node.GetLabels()[constants.NodeDeletionMark] == "true" {
		log.Info("Node should be removed due to GPUNode compaction, but it's not managed by TensorFusion, skip.", "name", node.Name)
	}

	if node.GetLabels()[constants.ProvisionerLabelKey] != "" {
		// Provision mode, match the provisionerID(GPUNode) here
		gpuNode := &tfv1.GPUNode{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: node.GetLabels()[constants.ProvisionerLabelKey]}, gpuNode); err != nil {
			return ctrl.Result{}, fmt.Errorf("get gpuNode(%s) : %w", node.GetLabels()[constants.ProvisionerLabelKey], err)
		}
		// set owned by GPUNode CR
		_ = controllerutil.SetControllerReference(gpuNode, node, r.Scheme)
		err := r.Client.Update(ctx, node)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("can not update node(%s) controller reference  : %w", node.Name, err)
		}

		// set GPU node's status to map to K8S node name
		gpuNode.Status.KubernetesNodeName = node.Name
		if err := r.Client.Status().Update(ctx, gpuNode); err != nil {
			return ctrl.Result{}, fmt.Errorf("can not update gpuNode(%s) status : %w", gpuNode.Name, err)
		}
	} else {
		// Select mode, GPU node is controlled by K8S node
		gpuNode := r.generateGPUNode(ctx, node, r.PoolState)

		// set owner reference to cascade delete
		e := controllerutil.SetControllerReference(node, gpuNode, r.Scheme)
		if e != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set controller reference: %w", e)
		}
		_, e = controllerutil.CreateOrPatch(ctx, r.Client, gpuNode, nil)
		if e != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create or patch GPUNode: %w", e)
		}
		log.Info("Created GPUNode due to selector matched", "name", gpuNode.Name)
	}

	return ctrl.Result{}, nil
}

func (r *NodeReconciler) generateGPUNode(ctx context.Context, node *corev1.Node, poolState config.GpuPoolState) *tfv1.GPUNode {
	poolName, err := poolState.GetMatchedPoolName(node)
	if err != nil {
		log.FromContext(ctx).Info("No matched GPU pool", "node", node.Name, "labels", node.Labels)
		return nil
	}
	gpuNode := &tfv1.GPUNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: node.Name,
			Labels: map[string]string{
				fmt.Sprintf(constants.GPUNodePoolIdentifierLabelFormat, poolName): "true",
			},
		},
		Spec: tfv1.GPUNodeSpec{
			ManageMode: tfv1.GPUNodeManageModeAutoSelect,
		},
		Status: tfv1.GPUNodeStatus{
			KubernetesNodeName: node.Name,
			ObservedGeneration: node.Generation,
		},
	}
	return gpuNode
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// must choose an initial label selector to avoid performance impact in large Kubernetes clusters
	selector := os.Getenv("INITIAL_GPU_NODE_LABEL_SELECTOR")
	if selector == "" {
		selector = constants.InitialGPUNodeSelector
	}
	selectors := strings.Split(selector, "=")
	p, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchLabels: map[string]string{
			selectors[0]: selectors[1],
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create predicate: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}, builder.WithPredicates(p)).
		Named("node").
		Complete(r)
	// When Pool changed, all nodes should re-generated, delete not matched ones
	//
}
