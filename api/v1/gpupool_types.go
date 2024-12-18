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
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GPUPoolSpec defines the desired state of GPUPool.
type GPUPoolSpec struct {
	CapacityConfig           CapacityConfig               `json:"capacityConfig,omitempty"`
	NodeManagerConfig        NodeManagerConfig            `json:"nodeManagerConfig,omitempty"`
	ObservabilityConfig      ObservabilityConfig          `json:"observabilityConfig,omitempty"`
	QosConfig                QosConfig                    `json:"qosConfig,omitempty"`
	ComponentConfig          ComponentConfig              `json:"componentConfig,omitempty"`
	SchedulingConfig         SchedulingConfigTemplateSpec `json:"schedulingConfig,omitempty"`
	SchedulingConfigTemplate string                       `json:"schedulingConfigTemplate,omitempty"`
}

type CapacityConfig struct {
	MinResources     GPUResourceUnit  `json:"minResources,omitempty"`
	MaxResources     GPUResourceUnit  `json:"maxResources,omitempty"`
	WarmResources    GPUResourceUnit  `json:"warmResources,omitempty"`
	Oversubscription Oversubscription `json:"oversubscription,omitempty"`
}

type Oversubscription struct {
	// the percentage of Host RAM appending to GPU VRAM, default to 50%
	VramExpandToHostMem string `json:"vramExpandToHostMem,omitempty"`

	// the percentage of Host Disk appending to GPU VRAM, default to 70%
	VramExpandToHostDisk string `json:"vramExpandToHostDisk,omitempty"`

	// The multipler of TFlops to oversell, default to 1 for production, 20 for development
	TflopsOversellRatio string `json:"tflopsOversellRatio,omitempty"`
}

type NodeManagerConfig struct {
	// karpenter mode Hypervisor manage GPU nodes and Workers
	NodeProvisioner             NodeProvisioner         `json:"nodeProvisioner,omitempty"`
	NodeSelector                NodeSelector            `json:"nodeSelector,omitempty"`
	NodeCompaction              NodeCompaction          `json:"nodeCompaction,omitempty"`
	NodePoolRollingUpdatePolicy NodeRollingUpdatePolicy `json:"nodePoolRollingUpdatePolicy,omitempty"`
}

// NodeProvisioner or NodeSelector, they are exclusive.
// NodeSelector is for existing GPUs, NodeProvisioner is for Karpenter-like auto management.
type NodeProvisioner struct {
	NodeClass    string        `json:"nodeClass,omitempty"`
	Requirements []Requirement `json:"requirements,omitempty"`
	Taints       []Taint       `json:"taints,omitempty"`
}

type Requirement struct {
	Key      string   `json:"key,omitempty"`
	Operator string   `json:"operator,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type Taint struct {
	Effect string `json:"effect,omitempty"`
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
}

// Use existing Kubernetes GPU nodes.
type NodeSelector []NodeSelectorItem

type NodeSelectorItem struct {
	MatchAny map[string]string `json:"matchAny,omitempty"`
	MatchAll map[string]string `json:"matchAll,omitempty"`
}

type NodeCompaction struct {
	Period string `json:"period,omitempty"`
}

type NodeRollingUpdatePolicy struct {
	// If set to false, updates will be pending in status, and user needs to manually approve updates.
	// Updates will occur immediately or during the next maintenance window.
	AutoUpdate      *bool  `json:"autoUpdate,omitempty"`
	BatchPercentage string `json:"batchPercentage,omitempty"`
	BatchInterval   string `json:"batchInterval,omitempty"`
	Duration        string `json:"duration,omitempty"`

	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
}

type MaintenanceWindow struct {
	// crontab syntax.
	Includes []string `json:"includes,omitempty"`
}

type ObservabilityConfig struct {
	Monitor MonitorConfig `json:"monitor,omitempty"`
	Alert   AlertConfig   `json:"alert,omitempty"`
}

type MonitorConfig struct {
	Interval string `json:"interval,omitempty"`
}

type AlertConfig struct {
	Expression runtime.RawExtension `json:"expression,omitempty"`
}

// Define different QoS and their price.
type QosConfig struct {
	Definitions   []QosDefinition `json:"definitions,omitempty"`
	DefaultQoS    string          `json:"defaultQoS,omitempty"`
	BillingPeriod string          `json:"billingPeriod,omitempty"` // "second" or "minute", default to "second"
	Pricing       []QosPricing    `json:"pricing,omitempty"`
}

type QosDefinition struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority,omitempty"` // Range from 1-100, reflects the scheduling priority when GPU is full and tasks are in the queue.
}

type GPUResourceUnit struct {
	// Tera floating point operations per second
	TFlops string `json:"tflops,omitempty"`

	// VRAM is short for Video memory, namely GPU RAM
	VRAM string `json:"vram,omitempty"`
}

type QosPricing struct {
	Qos                string          `json:"qos,omitempty"`
	Requests           GPUResourceUnit `json:"requests,omitempty"`
	LimitsOverRequests GPUResourceUnit `json:"limitsOverRequests,omitempty"`
}

// Customize system components for seamless onboarding.
type ComponentConfig struct {
	Worker     WorkerConfig     `json:"worker,omitempty"`
	Hypervisor HypervisorConfig `json:"hypervisor,omitempty"`
	Client     ClientConfig     `json:"client,omitempty"`
}

type WorkerConfig struct {
	Image             string               `json:"image,omitempty"` // "stable" | "latest" | "nightly"
	Port              int                  `json:"port,omitempty"`
	HostNetwork       *bool                `json:"hostNetwork,omitempty"`
	WorkerPodTemplate runtime.RawExtension `json:"workerPodTemplate,omitempty"` // Mixin extra spec.
}

type HypervisorConfig struct {
	Image                       string               `json:"image,omitempty"`
	HypervisorDaemonSetTemplate runtime.RawExtension `json:"hypervisorDaemonSetTemplate,omitempty"` // Mixin extra spec.
}

// TODO: client mutation webhook need TLS cert, need check using cert-manager or other ways
type ClientConfig struct {
	Image    string `json:"image,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Port     int    `json:"port,omitempty"`

	// +optional
	// define how to inject the client pod
	PodTemplateMergePatch runtime.RawExtension `json:"podTemplateMergePatch,omitempty"` // Add other things to the original pod.
}

// GPUPoolStatus defines the observed state of GPUPool.
type GPUPoolStatus struct {
	Cluster string `json:"cluster,omitempty"`

	Phase TensorFusionClusterPhase `json:"phase,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`

	TotalNodes    int32 `json:"totalNodes,omitempty"`
	TotalGPUs     int32 `json:"totalGPUs,omitempty"`
	ReadyNodes    int32 `json:"readyNodes,omitempty"`
	NotReadyNodes int32 `json:"notReadyNodes,omitempty"`

	TotalTFlops int32  `json:"totalTFlops,omitempty"`
	TotalVRAM   string `json:"totalVRAM,omitempty"`

	AvailableTFlops int32  `json:"availableTFlops,omitempty"`
	AvailableVRAM   string `json:"availableVRAM,omitempty"`

	// If using provisioner, GPU nodes could be outside of the K8S cluster.
	// The GPUNodes custom resource will be created and deleted automatically.
	// ProvisioningStatus is to track the status of those outside GPU nodes.
	ProvisioningStatus PoolProvisioningStatus `json:"provisioningStatus,omitempty"`

	// when updating any component version or config, poolcontroller will perform rolling update.
	// the status will be updated periodically, default to 5s, progress will be 0-100.
	// when the progress is 100, the component version or config is fully updated.
	ComponentStatus PoolComponentStatus `json:"componentStatus,omitempty"`
}

type PoolProvisioningStatus struct {
	InitializingNodes int32 `json:"initializingNodes,omitempty"`
	TerminatingNodes  int32 `json:"terminatingNodes,omitempty"`
	AvailableNodes    int32 `json:"availableNodes,omitempty"`
}

type PoolComponentStatus struct {
	WorkerVersion        string `json:"worker,omitempty"`
	WorkerConfigSynced   bool   `json:"workerConfigSynced,omitempty"`
	WorkerUpdateProgress int32  `json:"workerUpdateProgress,omitempty"`

	HypervisorVersion        string `json:"hypervisor,omitempty"`
	HypervisorConfigSynced   bool   `json:"hypervisorConfigSynced,omitempty"`
	HyperVisorUpdateProgress int32  `json:"hypervisorUpdateProgress,omitempty"`

	ClientVersion        string `json:"client,omitempty"`
	ClientConfigSynced   bool   `json:"clientConfigSynced,omitempty"`
	ClientUpdateProgress int32  `json:"clientUpdateProgress,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// GPUPool is the Schema for the gpupools API.
type GPUPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GPUPoolSpec   `json:"spec,omitempty"`
	Status GPUPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GPUPoolList contains a list of GPUPool.
type GPUPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GPUPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GPUPool{}, &GPUPoolList{})
}
