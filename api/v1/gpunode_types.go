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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GPUNodeSpec defines the desired state of GPUNode.
type GPUNodeSpec struct {
	ManageMode GPUNodeManageMode `json:"manageMode,omitempty"`

	// if not all GPU cards should be used, specify the GPU card indices, default to empty,
	// onboard all GPU cards to the pool
	GPUCardIndices []int `json:"gpuCardIndices,omitempty"`
}

type GPUNodeManageMode string

const (
	GPUNodeManageModeNone   GPUNodeManageMode = "manual"
	GPUNodeManageModeAuto   GPUNodeManageMode = "selected"
	GPUNodeManageModeManual GPUNodeManageMode = "provisioned"
)

// GPUNodeStatus defines the observed state of GPUNode.
type GPUNodeStatus struct {
	Phase TensorFusionClusterPhase `json:"phase,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`

	TotalTFlops int32  `json:"totalTFlops,omitempty"`
	TotalVRAM   string `json:"totalVRAM,omitempty"`

	AvailableTFlops int32  `json:"availableTFlops,omitempty"`
	AvailableVRAM   string `json:"availableVRAM,omitempty"`

	HypervisorStatus NodeHypervisorStatus `json:"hypervisorStatus,omitempty"`

	NodeInfo GPUNodeInfo `json:"nodeInfo,omitempty"`

	LoadedModels []string `json:"loadedModels,omitempty"`

	TotalGPUs             int32    `json:"totalGPUs,omitempty"`
	ManagedGPUs           int32    `json:"managedGPUs,omitempty"`
	ManagedGPUResourceIDs []string `json:"managedGPUResourceIDs,omitempty"`
}

type GPUNodeInfo struct {
	Hostname         string `json:"hostname,omitempty"`
	IP               string `json:"ip,omitempty"`
	KernalVersion    string `json:"kernalVersion,omitempty"`
	OSImage          string `json:"osImage,omitempty"`
	GPUDriverVersion string `json:"gpuDriverVersion,omitempty"`
	GPUModel         string `json:"gpuModel,omitempty"`
	GPUCount         int32  `json:"gpuCount,omitempty"`
	OperatingSystem  string `json:"operatingSystem,omitempty"`
	Architecture     string `json:"architecture,omitempty"`
}

type NodeHypervisorStatus struct {
	HypervisorState   string      `json:"hypervisorState,omitempty"`
	HypervisorVersion string      `json:"hypervisorVersion,omitempty"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// GPUNode is the Schema for the gpunodes API.
type GPUNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GPUNodeSpec   `json:"spec,omitempty"`
	Status GPUNodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GPUNodeList contains a list of GPUNode.
type GPUNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GPUNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GPUNode{}, &GPUNodeList{})
}
