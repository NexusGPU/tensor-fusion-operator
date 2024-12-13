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

// TensorFusionClusterSpec defines the desired state of TensorFusionCluster.
type TensorFusionClusterSpec struct {
	Enroll EnrollConfig `json:"enroll,omitempty"`

	GPUPools []GPUPoolDefinition `json:"gpuPools,omitempty"`

	// +optional
	ComputingVendor ComputingVendorConfig `json:"computingVendor,omitempty"`

	// +optional
	StorageVendor StorageVendorConfig `json:"storageVendor,omitempty"`

	// +optional
	DataPipelines DataPipelinesConfig `json:"dataPipelines,omitempty"`
}

// TensorFusionClusterStatus defines the observed state of TensorFusionCluster.
type TensorFusionClusterStatus struct {

	// +kubebuilder:default:=Initializing
	Phase TensorFusionClusterPhase `json:"phase,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`

	TotalPools int32 `json:"totalPools,omitempty"`
	TotalNodes int32 `json:"totalNodes,omitempty"`
	TotalGPUs  int32 `json:"totalGPUs,omitempty"`

	TotalTFlops int32  `json:"totalTFlops,omitempty"`
	TotalVRAM   string `json:"totalVRAM,omitempty"`

	AvailableTFlops int32  `json:"availableTFlops,omitempty"`
	AvailableVRAM   string `json:"availableVRAM,omitempty"`

	ReadyGPUPools    []string `json:"readyGPUPools,omitempty"`
	NotReadyGPUPools []string `json:"notReadyGPUPools,omitempty"`

	AvailableLicenses  int32       `json:"availableLicenses,omitempty"`
	TotalLicenses      int32       `json:"totalLicenses,omitempty"`
	LicenseRenewalTime metav1.Time `json:"licenseRenewalTime,omitempty"`

	CloudConnectionStatus ClusterCloudConnectionStatus `json:"cloudConnectionStatus,omitempty"`
	StorageStatus         ClusterStorageStatus         `json:"storageStatus,omitempty"`
	ComputingVendorStatus ClusterComputingVendorStatus `json:"computingVendorStatus,omitempty"`
}

type ClusterCloudConnectionStatus struct {
	ClusterID         string      `json:"clusterId,omitempty"`
	ConnectionState   string      `json:"connectionState,omitempty"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
}

type ClusterStorageStatus struct {
	ConnectionState string `json:"connectionState,omitempty"`
}

type ClusterComputingVendorStatus struct {
	ConnectionState string `json:"connectionState,omitempty"`
}

// TensorFusionClusterPhase represents the phase of the TensorFusionCluster resource.
type TensorFusionClusterPhase string

const (
	TensorFusionClusterInitializing = TensorFusionClusterPhase("Initializing")
	TensorFusionClusterRunning      = TensorFusionClusterPhase("Running")
	TensorFusionClusterUpdating     = TensorFusionClusterPhase("Updating")
	TensorFusionClusterDestroying   = TensorFusionClusterPhase("Destroying")
)

// Enroll to TensorFusion cloud with a enrollment key
type EnrollConfig struct {
	APIEndpoint string              `json:"apiEndpoint,omitempty"` // API endpoint for enrollment.
	EnrollKey   EnrollmentKeyConfig `json:"enrollKey,omitempty"`
}

type EnrollmentKeyConfig struct {
	Data      string        `json:"data,omitempty"` // Enrollment key data.
	SecretRef NameNamespace `json:"secretRef,omitempty"`
}

// GPUPool defines how to create a GPU pool, could be URL or inline
type GPUPoolDefinition struct {
	Name string `json:"name,omitempty"` // Name of the GPU pool.

	// +optional
	Spec GPUPoolSpec `json:"spec,omitempty"`

	// +optional
	SpecTemplateURL string `json:"specTemplateUrl,omitempty"`
}

// ComputingVendorConfig defines the Cloud vendor connection such as AWS, GCP, Azure etc.
type ComputingVendorConfig struct {
	Name     string `json:"name,omitempty"`     // Name of the computing vendor.
	Type     string `json:"type,omitempty"`     // Type of the computing vendor (e.g., aws, lambdalabs, gcp, azure, together.ai).
	AuthType string `json:"authType,omitempty"` // Authentication type (e.g., accessKey, serviceAccount).

	// +optional
	Enable *bool `json:"enable,omitempty"` // Enable or disable the computing vendor.

	GPUNodeControllerType string                `json:"gpuNodeControllerType,omitempty"` // Type of GPU node controller (e.g., asg, karpenter, native).
	Params                ComputingVendorParams `json:"params,omitempty"`
}

type ComputingVendorParams struct {
	Region    string `json:"region,omitempty"`    // Region for the computing vendor.
	AccessKey string `json:"accessKey,omitempty"` // Access key for the computing vendor.
	SecretKey string `json:"secretKey,omitempty"` // Secret key for the computing vendor.
	IAMRole   string `json:"iamRole,omitempty"`   // IAM role for the computing vendor like AWS
}

// StorageVendorConfig defines Postgres database with extensions for timeseries storage and other resource aggregation results, system events and diagnostics reports etc.
type StorageVendorConfig struct {
	Mode  string `json:"mode,omitempty"`  // Mode of the storage vendor (e.g., cloudnative-pg, timescale-db, RDS for PG).
	Image string `json:"image,omitempty"` // Image for the storage vendor (default to timescale).

	// +optional
	InstallCloudNativePGOperator *bool `json:"installCloudNativePGOperator,omitempty"` // Whether to install CloudNative-PG operator.

	StorageClass      string               `json:"storageClass,omitempty"`      // Storage class for the storage vendor.
	PGExtensions      []string             `json:"pgExtensions,omitempty"`      // List of PostgreSQL extensions to install.
	PGClusterTemplate runtime.RawExtension `json:"pgClusterTemplate,omitempty"` // Extra spec for the PostgreSQL cluster template.
}

// DataPipelinesConfig defines the aggregation jobs that can make statistics on the data and then report to cloud if configured.
type DataPipelinesConfig struct {
	Resources DataPipeline4ResourcesConfig `json:"resources,omitempty"`

	Timeseries DataPipeline4TimeSeriesConfig `json:"timeseries,omitempty"`
}

type DataPipeline4ResourcesConfig struct {
	// +optional
	SyncToCloud *bool `json:"syncToCloud,omitempty"` // Whether to sync resources to the cloud.

	// +optional human readable time like 1h, 1d, default to 1h
	SyncPeriod string `json:"syncPeriod,omitempty"` // Period for syncing resources.
}

type DataPipeline4TimeSeriesConfig struct {
	AggregationPeriods       []string          `json:"aggregationPeriods,omitempty"`       // List of aggregation periods.
	RawDataRetention         string            `json:"rawDataRetention,omitempty"`         // Retention period for raw data.
	AggregationDataRetention string            `json:"aggregationDataRetention,omitempty"` // Retention period for aggregated data.
	RemoteWrite              RemoteWriteConfig `json:"remoteWrite,omitempty"`              // Configuration for remote write.
}

// RemoteWriteConfig represents the configuration for remote write.
type RemoteWriteConfig struct {
	Connection DataPipelineResultRemoteWriteConfig `json:"connection,omitempty"`
	Metrics    []string                            `json:"metrics,omitempty"` // List of metrics to remote write.
}

type DataPipelineResultRemoteWriteConfig struct {
	Type string `json:"type,omitempty"` // Type of the connection (e.g., datadog).
	URL  string `json:"url,omitempty"`  // URL of the connection.
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// TensorFusionCluster is the Schema for the tensorfusionclusters API.
type TensorFusionCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TensorFusionClusterSpec   `json:"spec,omitempty"`
	Status TensorFusionClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TensorFusionClusterList contains a list of TensorFusionCluster.
type TensorFusionClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TensorFusionCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TensorFusionCluster{}, &TensorFusionClusterList{})
}
