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

package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
)

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager, config *config.PodMutator) error {
	webhookServer := mgr.GetWebhookServer()
	webhookServer.Register("/mutate-v1-pod",
		&admission.Webhook{
			Handler: &TensorFusionPodMutator{
				Config: config,
				Client: mgr.GetClient(),
			},
		})
	return nil
}

type TensorFusionPodMutator struct {
	Client  client.Client
	Config  *config.PodMutator
	decoder admission.Decoder
}

// Handle implements admission.Handler interface.
func (m *TensorFusionPodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := m.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log := log.FromContext(ctx)
	log.Info("Mutating pod", "name", pod.Name, "namespace", pod.Namespace)

	reqs := parseTFReq(pod)
	if len(reqs) == 0 {
		return admission.Allowed("no tensor fusion requirements found")
	}

	// 1. Inject initContainer and env variables
	patches, err := m.patchTFClient(pod, reqs)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// generate tensor fusion connections and apply to cluster
	tfConnections := generateTensorFusionConnection(pod, reqs)

	for _, tfConnection := range tfConnections {
		if err := m.Client.Create(ctx, tfConnection); err != nil {
			log.Error(err, "Failed to create TensorFusionConnection")
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	return admission.Patched("tensor fusion component patched", patches...)
}

// InjectDecoder injects the decoder.
func (m *TensorFusionPodMutator) InjectDecoder(d admission.Decoder) error {
	m.decoder = d
	return nil
}

type TFReq struct {
	ContainerName string
	Tflops        resource.Quantity
	Vram          resource.Quantity
}

func parseTFReq(pod *corev1.Pod) []TFReq {
	if pod.Annotations == nil {
		return nil
	}

	reqs := make([]TFReq, 0, len(pod.Spec.Containers))

	for _, container := range pod.Spec.Containers {
		containerName := container.Name

		// Check if TF requirements exist for this container
		tflopsKey := fmt.Sprintf(constants.TFLOPSContainerAnnotationFormat, containerName)
		vramKey := fmt.Sprintf(constants.VRAMContainerAnnotationFormat, containerName)

		tflopsStr, hasTflops := pod.Annotations[tflopsKey]
		vramStr, hasVram := pod.Annotations[vramKey]

		if !hasTflops && !hasVram {
			continue
		}

		req := TFReq{
			ContainerName: containerName,
		}

		// Parse TFLOPS requirement
		if hasTflops {
			tflops, err := resource.ParseQuantity(tflopsStr)
			if err == nil {
				req.Tflops = tflops
			}
		}

		// Parse VRAM requirement
		if hasVram {
			vram, err := resource.ParseQuantity(vramStr)
			if err == nil {
				req.Vram = vram
			}
		}

		reqs = append(reqs, req)
	}

	return reqs
}

func (m *TensorFusionPodMutator) patchTFClient(pod *corev1.Pod, tfReq []TFReq) ([]jsonpatch.JsonPatchOperation, error) {
	podPatch := m.Config.PatchStrategicMerge
	// Copy containers
	podPatch.Spec.Containers = append([]corev1.Container{}, podPatch.Spec.Containers...)

	// Patch env vars
	for _, req := range tfReq {
		for _, container := range podPatch.Spec.Containers {
			if container.Name == req.ContainerName {
				container.Env = append(container.Env, m.Config.PatchEnvVars...)
			}
		}
	}

	// Convert the strategic merge patch to JSON
	patchBytes, err := json.Marshal(m.Config.PatchStrategicMerge)
	if err != nil {
		return nil, fmt.Errorf("marshal patch: %v", err)
	}

	// Convert the current pod to JSON
	currentBytes, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("marshal current pod: %v", err)
	}

	// Apply the strategic merge patch
	resultBytes, err := strategicpatch.StrategicMergePatch(currentBytes, patchBytes, corev1.Pod{})
	if err != nil {
		return nil, fmt.Errorf("apply strategic merge patch: %v", err)
	}

	// Generate JSON patch operations by comparing original and patched pod
	patches, err := jsonpatch.CreatePatch(currentBytes, resultBytes)
	if err != nil {
		return nil, fmt.Errorf("create json patch: %v", err)
	}

	// Unmarshal the result back into the pod
	if err := json.Unmarshal(resultBytes, pod); err != nil {
		return nil, fmt.Errorf("unmarshal patched pod: %v", err)
	}

	return patches, nil
}

func generateTensorFusionConnection(pod *corev1.Pod, tfReq []TFReq) []*tfv1.TensorFusionConnection {
	connections := make([]*tfv1.TensorFusionConnection, 0, len(tfReq))

	for _, req := range tfReq {
		connection := &tfv1.TensorFusionConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-tf-%s", pod.Name, req.ContainerName),
				Namespace: pod.Namespace,
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
				Resources: tfv1.Resources{
					Requests: tfv1.Resource{
						Tflops: req.Tflops,
						Vram:   req.Vram,
					},
					Limits: tfv1.Resource{
						Tflops: req.Tflops,
						Vram:   req.Vram,
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
