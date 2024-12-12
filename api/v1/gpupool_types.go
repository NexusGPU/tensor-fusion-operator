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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GPUPoolSpec defines the desired state of GPUPool.
type GPUPoolSpec struct {
	NodeManagerConfig   NodeManagerConfig   `json:"nodeManagerConfig,omitempty"`
	ObservabilityConfig ObservabilityConfig `json:"observabilityConfig,omitempty"`
	QosConfig           QosConfig           `json:"qosConfig,omitempty"`
	ComponentConfig     ComponentConfig     `json:"componentConfig,omitempty"`
	SchedulingConfig    SchedulingConfig    `json:"schedulingConfig,omitempty"`
}

type NodeManagerConfig struct {
	NodeProvisioner         NodeProvisioner         `json:"nodeProvisioner,omitempty"`
	NodeSelector            NodeSelector            `json:"nodeSelector,omitempty"`
	NodeCompaction          NodeCompaction          `json:"nodeCompaction,omitempty"`
	NodeRollingUpdatePolicy NodeRollingUpdatePolicy `json:"nodeRollingUpdatePolicy,omitempty"`
}

// NodeProvisioner or NodeSelector, they are exclusive.
// NodeSelector is for existing GPUs, NodeProvisioner is for Karpenter-like auto management.
type NodeProvisioner struct {
	NodeClass    string         `json:"nodeClass,omitempty"`
	Requirements []Requirement  `json:"requirements,omitempty"`
	Resources    map[string]any `json:"resources,omitempty"`
	Taints       []Taint        `json:"taints,omitempty"`
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
	Period time.Duration `json:"period,omitempty"`
}

type NodeRollingUpdatePolicy struct {
	// If set to false, updates will be pending in status, and user needs to manually approve updates.
	// Updates will occur immediately or during the next maintenance window.
	AutoUpdate      bool          `json:"autoUpdate,omitempty"`
	BatchPercentage string        `json:"batchPercentage,omitempty"`
	BatchInterval   time.Duration `json:"batchInterval,omitempty"`
	Duration        time.Duration `json:"duration,omitempty"`

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
	Interval time.Duration `json:"interval,omitempty"`
}

type AlertConfig struct {
	Expression map[string]any `json:"expression,omitempty"`
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

type QosPricing struct {
	Qos                string            `json:"qos,omitempty"`
	Requests           map[string]string `json:"requests,omitempty"`
	LimitsOverRequests map[string]string `json:"limitsOverRequests,omitempty"`
}

// Customize system components for seamless onboarding.
type ComponentConfig struct {
	Worker     WorkerConfig     `json:"worker,omitempty"`
	Hypervisor HypervisorConfig `json:"hypervisor,omitempty"`
	Client     ClientConfig     `json:"client,omitempty"`
}

type WorkerConfig struct {
	Image             string         `json:"image,omitempty"` // "stable" | "latest" | "nightly"
	Port              int            `json:"port,omitempty"`
	HostNetwork       bool           `json:"hostNetwork,omitempty"`
	WorkerPodTemplate map[string]any `json:"workerPodTemplate,omitempty"` // Mixin extra spec.
}

type HypervisorConfig struct {
	Image                       string         `json:"image,omitempty"`
	HypervisorDaemonSetTemplate map[string]any `json:"hypervisorDaemonSetTemplate,omitempty"` // Mixin extra spec.
}

type ClientConfig struct {
	Image                                string                               `json:"image,omitempty"` // "stable" | "latest" | "nightly", must match worker side.
	Protocol                             string                               `json:"protocol,omitempty"`
	Port                                 int                                  `json:"port,omitempty"`
	ChangeOrRemoveOriginalNodeAffinities ChangeOrRemoveOriginalNodeAffinities `json:"changeOrRemoveOriginalNodeAffinities,omitempty"`
	ChangeOrRemoveOriginalTolerations    ChangeOrRemoveOriginalTolerations    `json:"changeOrRemoveOriginalTolerations,omitempty"`
	RemoveOriginalGPUResources           RemoveOriginalGPUResources           `json:"removeOriginalGPUResources,omitempty"`
	InjectContainerTemplate              map[string]any                       `json:"injectContainerTemplate,omitempty"`
	PodTemplatePatch                     map[string]any                       `json:"podTemplatePatch,omitempty"` // Add other things to the original pod.
}

type ChangeOrRemoveOriginalNodeAffinities struct {
	Enable           bool              `json:"enable,omitempty"`
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty"`
}

type ChangeOrRemoveOriginalTolerations struct {
	Enable           bool              `json:"enable,omitempty"`
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty"`
}

type RemoveOriginalGPUResources struct {
	Enable            bool     `json:"enable,omitempty"`
	MatchResourceKeys []string `json:"matchResourceKeys,omitempty"`
}

type MatchExpression struct {
	Key       string   `json:"key,omitempty"`
	Values    []string `json:"values,omitempty"`
	NewValues []string `json:"newValues,omitempty"` // If new values are empty, indicates this key should be deleted.
}

// Place the workload and scale smart.
type SchedulingConfig struct {

	// place the client or worker to best matched nodes
	Placement PlacementConfig `json:"placement,omitempty"`

	// scale the workload based on the usage and traffic
	AutoScaling AutoScalingConfig `json:"autoScaling,omitempty"`

	// avoid hot GPU devices and continuously balance the workload
	// implemented by trigger a simulation scheduling and advise better GPU nodes for scheduler
	ReBalancer ReBalancerConfig `json:"reBalancer,omitempty"`

	// single GPU device multi-process queuing and fair scheduling with QoS constraint
	Hypervisor HypervisorScheduling `json:"hypervisor,omitempty"`
}

type PlacementConfig struct {
	Mode               PlacementMode `json:"mode,omitempty"`               // "compactFirst" | "lowLoadFirst"
	AllowUsingLocalGPU bool          `json:"allowUsingLocalGPU,omitempty"` // If false, workloads will not be scheduled directly to GPU nodes with 'localGPU: true'.
	GPUFilters         []GPUFilter   `json:"gpuFilters,omitempty"`
}

type PlacementMode string

const (
	// default to compactFirst for cost saving and energy saving
	PlacementModeCompactFirst PlacementMode = "compactFirst"

	// in some cases, use lowLoadFirst for balance and fairness
	PlacementModeLowLoadFirst PlacementMode = "lowLoadFirst"
)

// GPUFilter is to select eligible GPUs for scheduling.
//
// example:
// ```yaml
// - type: avoidTooMuchConnectionsOnSameGPU
// params:
//
//	connectionNum: 150
//
// - type: avoidDifferentZone
// params:
//
//	# by default, GPU worker will be scheduled into the same zone as CPU Client Pod to align AZ and improve performance
//	topologyKey: topology.kubernetes.io/zone
//
// ```
type GPUFilter struct {
	Type   string         `json:"type,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

type AutoScalingConfig struct {
	// layer 1 vertical auto-scaling, turbo burst to existing GPU cards fastly
	AutoSetLimits AutoSetLimits `json:"autoSetLimits,omitempty"`

	// layer 2 horizontal auto-scaling, scale up to more GPU cards if max limits threshod hit
	AutoSetReplicas AutoSetReplicas `json:"autoSetReplicas,omitempty"`

	// layer 3 adjusting, to match the actual usage in the long run
	AutoSetRequests AutoSetRequests `json:"autoSetRequests,omitempty"`

	// additional layer to save VRAM, auto-freeze memory and cool down to RAM and Disk
	ScaleToZero ScaleToZero `json:"scaleToZero,omitempty"`
}

// A typical autoLimits algorithm could be checking every 5m, look back 1 day data,
// select 99% of actual usage as preferredLimits,
// calculate finalPreferredLimits, which is preferredLimits*(1+extraBufferRatio)
// if they are equal with each other within a range (eg. 5%), do nothing
// if finalPreferredLimits is less than current limits and exceeded error range,
// set current limits to finalPreferredLimits
// if finalPreferredLimits > current limits and exceeded error range,
// set current limits to max(finalPreferredLimits, current limits * scaleUpStep)
// if AI prediction enabled, it helps to detect history pattern, and set more reasonable, explainable limit value
// the final set limits should be max(finalPreferredLimits, last(predict_value * (1 + extraTFlopsBufferRatio)))
type AutoSetLimits struct {
	EvaluationPeriod       time.Duration `json:"evaluationPeriod,omitempty"`
	ExtraTFlopsBufferRatio string        `json:"extraTFlopsBufferRatio,omitempty"`
	IgnoredDeltaRange      string        `json:"ignoredDeltaRange,omitempty"`
	ScaleUpStep            string        `json:"scaleUpStep,omitempty"`
	MaxRatioToRequests     float64       `json:"maxRatioToRequests,omitempty"`
	Prediction             Prediction    `json:"prediction,omitempty"`
}

// To handle burst traffic, scale up in short time (this feature requires GPU context migration & replication, not available yet)
type AutoSetReplicas struct {
	Enable                bool          `json:"enable,omitempty"`
	TargetTFlopsOfLimits  string        `json:"targetTFlopsOfLimits,omitempty"`
	EvaluationPeriod      time.Duration `json:"evaluationPeriod,omitempty"`
	ScaleUpStep           string        `json:"scaleUpStep,omitempty"`
	ScaleDownStep         string        `json:"scaleDownStep,omitempty"`
	ScaleDownUpDownTime   time.Duration `json:"scaleDownUpDownTime,omitempty"`
	ScaleDownCoolDownTime time.Duration `json:"scaleDownCoolDownTime,omitempty"`
}

type AutoSetRequests struct {
	PercentileForAutoRequests string        `json:"percentileForAutoRequests,omitempty"`
	ExtraBufferRatio          float64       `json:"extraBufferRatio,omitempty"`
	EvaluationPeriod          time.Duration `json:"evaluationPeriod,omitempty"`
	AggregationPeriod         time.Duration `json:"aggregationPeriod,omitempty"`
	Prediction                Prediction    `json:"prediction,omitempty"`
}

type ScaleToZero struct {
	AutoFreeze         []AutoFreeze       `json:"autoFreeze,omitempty"`
	IntelligenceWarmup IntelligenceWarmup `json:"intelligenceWarmup,omitempty"`
}

type AutoFreeze struct {
	Qos             string        `json:"qos,omitempty"`
	FreezeToMemTTL  time.Duration `json:"freezeToMemTTL,omitempty"`
	FreezeToDiskTTL time.Duration `json:"freezeToDiskTTL,omitempty"`
	Enable          bool          `json:"enable,omitempty"`
}

type IntelligenceWarmup struct {
	Enable                  bool          `json:"enable,omitempty"`
	Model                   string        `json:"model,omitempty"`
	IngestHistoryDataPeriod time.Duration `json:"ingestHistoryDataPeriod,omitempty"`
	PredictionPeriod        time.Duration `json:"predictionPeriod,omitempty"`
}

type Prediction struct {
	Enable                  bool          `json:"enable,omitempty"`
	Model                   string        `json:"model,omitempty"`
	IngestHistoryDataPeriod time.Duration `json:"ingestHistoryDataPeriod,omitempty"`
	PredictionPeriod        time.Duration `json:"predictionPeriod,omitempty"`
}

type ReBalancerConfig struct {
	Internal              time.Duration `json:"internal,omitempty"`
	ReBalanceCoolDownTime time.Duration `json:"reBalanceCoolDownTime,omitempty"`
	Threshold             Threshold     `json:"threshold,omitempty"`
}

type Threshold struct {
	MatchAny map[string]any `json:"matchAny,omitempty"`
}

type HypervisorScheduling struct {
	MultiProcessQueuing MultiProcessQueuing `json:"multiProcessQueuing,omitempty"`
}

type MultiProcessQueuing struct {
	Enable               bool          `json:"enable,omitempty"`
	Interval             time.Duration `json:"interval,omitempty"`
	QueueLevelTimeSlices []string      `json:"queueLevelTimeSlices,omitempty"`
}

// GPUPoolStatus defines the observed state of GPUPool.
type GPUPoolStatus struct {
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
	ProvisioningStatus struct {
		InitializingNodes int32 `json:"initializingNodes,omitempty"`
		TerminatingNodes  int32 `json:"terminatingNodes,omitempty"`
		AvailableNodes    int32 `json:"availableNodes,omitempty"`
	} `json:"provisioningStatus,omitempty"`

	// when updating any component version or config, poolcontroller will perform rolling update.
	// the status will be updated periodically, default to 5s, progress will be 0-100.
	// when the progress is 100, the component version or config is fully updated.
	ComponentStatus struct {
		WorkerVersion        string `json:"worker,omitempty"`
		WorkerConfigSynced   bool   `json:"workerConfigSynced,omitempty"`
		WorkerUpdateProgress int32  `json:"workerUpdateProgress,omitempty"`

		HypervisorVersion        string `json:"hypervisor,omitempty"`
		HypervisorConfigSynced   bool   `json:"hypervisorConfigSynced,omitempty"`
		HyperVisorUpdateProgress int32  `json:"hypervisorUpdateProgress,omitempty"`

		ClientVersion        string `json:"client,omitempty"`
		ClientConfigSynced   bool   `json:"clientConfigSynced,omitempty"`
		ClientUpdateProgress int32  `json:"clientUpdateProgress,omitempty"`
	} `json:"componentStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
