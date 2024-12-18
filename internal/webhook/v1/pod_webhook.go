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
	"path"

	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	"github.com/lithammer/shortuuid/v4"
	"github.com/samber/lo"
)

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager, config *config.PodMutation) error {
	webhookServer := mgr.GetWebhookServer()

	webhookServer.Register("/mutate-v1-pod",
		&admission.Webhook{
			Handler: &TensorFusionPodMutator{
				Config:  config,
				decoder: admission.NewDecoder(runtime.NewScheme()),
				Client:  mgr.GetClient(),
			},
		})
	return nil
}

type TensorFusionPodMutator struct {
	Client  client.Client
	Config  *config.PodMutation
	decoder admission.Decoder
}

// Handle implements admission.Handler interface.
func (m *TensorFusionPodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := m.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log := log.FromContext(ctx)
	log.Info("Mutating pod", "generateName", pod.GenerateName, "namespace", pod.Namespace)

	reqs := ParseTFReq(pod)
	if len(reqs) == 0 {
		return admission.Allowed("no tensor fusion requirements found")
	}

	// 1. Inject initContainer and env variables
	patches, err := m.patchTFClient(pod, reqs)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.Patched("tensor fusion component patched", patches...)
}

// InjectDecoder injects the decoder.
func (m *TensorFusionPodMutator) InjectDecoder(d admission.Decoder) error {
	m.decoder = d
	return nil
}

type TFReq struct {
	ContainerName       string
	ConnectionName      string
	ConnectionNamespace string
	Tflops              resource.Quantity
	Vram                resource.Quantity
}

func ParseTFReq(pod *corev1.Pod) []TFReq {
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

		connectionNameEnv, ok := lo.Find(container.Env, func(e corev1.EnvVar) bool {
			return e.Name == constants.ConnectionNameEnv
		})

		if !ok {
			continue
		}
		connectionNamespaceEnv, ok := lo.Find(container.Env, func(e corev1.EnvVar) bool {
			return e.Name == constants.ConnectionNamespaceEnv
		})

		if !ok {
			continue
		}

		req := TFReq{
			ContainerName:       containerName,
			ConnectionName:      connectionNameEnv.Name,
			ConnectionNamespace: connectionNamespaceEnv.Name,
		}
		connectionNameEnv, ok := lo.Find(container.Env, func(e corev1.EnvVar) bool {
			return e.Name == constants.ConnectionNameEnv
		})
		if ok {
			req.ConnectionName = connectionNameEnv.Name
		}
		connectionNamespaceEnv, ok := lo.Find(container.Env, func(e corev1.EnvVar) bool {
			return e.Name == constants.ConnectionNamespaceEnv
		})
		if ok {
			req.ConnectionNamespace = connectionNamespaceEnv.Name
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
	// Convert the current pod to JSON
	currentBytes, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("marshal current pod: %v", err)
	}

	// Patch to Container
	for _, req := range tfReq {
		for i := range pod.Spec.Containers {
			container := &pod.Spec.Containers[i]
			if container.Name == req.ContainerName {
				// patch from config
				containerJSON, err := json.Marshal(container)
				if err != nil {
					return nil, fmt.Errorf("marshal container: %v", err)
				}
				patchJSON, err := json.Marshal(m.Config.PatchToContainer)
				if err != nil {
					return nil, fmt.Errorf("marshal patchToContainer: %v", err)
				}

				patchedJSON, err := strategicpatch.StrategicMergePatch(containerJSON, patchJSON, corev1.Container{})
				if err != nil {
					return nil, fmt.Errorf("apply strategic merge patch to container: %v", err)
				}
				if err := json.Unmarshal(patchedJSON, container); err != nil {
					return nil, fmt.Errorf("unmarshal patched container: %v", err)
				}

				// add connection env
				connectionName := fmt.Sprintf("%s-tf-worker-%s", pod.GenerateName+container.Name, shortuuid.New())
				connectionNamespace := pod.Namespace

				container.Env = append(container.Env, corev1.EnvVar{
					Name:  constants.ConnectionNameEnv,
					Value: connectionName,
				})
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  constants.ConnectionNamespaceEnv,
					Value: connectionNamespace,
				})
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  constants.GetConnectionURLEnv,
					Value: path.Join(m.Config.OperatorEndpoint, fmt.Sprintf("/api/connection?name=%s,namespace=%s", connectionName, connectionNamespace)),
				})
			}
		}
	}

	containerPatchedJSON, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("marshal current pod: %v", err)
	}
	patches, err := jsonpatch.CreatePatch(currentBytes, containerPatchedJSON)
	if err != nil {
		return nil, fmt.Errorf("patch to container: %v", err)
	}

	// Convert the strategic merge patch to JSON
	patchBytes, err := json.Marshal(m.Config.PatchToPod)

	if err != nil {
		return nil, fmt.Errorf("marshal patch: %v", err)
	}

	// Apply the strategic merge patch
	resultBytes, err := strategicpatch.StrategicMergePatch(currentBytes, patchBytes, corev1.Pod{})
	if err != nil {
		return nil, fmt.Errorf("apply strategic merge patch: %v", err)
	}

	// Generate JSON patch operations by comparing original and patched pod
	strategicpatches, err := jsonpatch.CreatePatch(currentBytes, resultBytes)
	if err != nil {
		return nil, fmt.Errorf("create json patch: %v", err)
	}

	// Unmarshal the result back into the pod
	if err := json.Unmarshal(resultBytes, pod); err != nil {
		return nil, fmt.Errorf("unmarshal patched pod: %v", err)
	}

	patches = append(patches, strategicpatches...)
	return patches, nil
}
