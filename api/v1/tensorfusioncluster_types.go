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

// TensorFusionClusterSpec defines the desired state of TensorFusionCluster.
type TensorFusionClusterSpec struct {
	Enroll EnrollConfig `json:"enroll,omitempty"`

	GPUPools []GPUPoolDefinition `json:"gpuPools,omitempty"`

	ComputingVendors []ComputingVendorConfig `json:"computingVendors,omitempty"`

	StorageVendors []StorageVendorConfig `json:"storageVendors,omitempty"`

	DataPipelines DataPipelinesConfig `json:"dataPipelines,omitempty"`
}

// TensorFusionClusterStatus defines the observed state of TensorFusionCluster.
type TensorFusionClusterStatus struct {
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

	CloudConnectionStatus struct {
		ClusterID         string      `json:"clusterId,omitempty"`
		ConnectionState   string      `json:"connectionState,omitempty"`
		LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	} `json:"cloudConnectionStatus,omitempty"`

	StorageStatus struct {
		ConnectionState string `json:"connectionState,omitempty"`
	}

	ComputingVendorStatus struct {
		ConnectionState string `json:"connectionState,omitempty"`
	}
}

// TensorFusionClusterPhase represents the phase of the TensorFusionCluster resource.
type TensorFusionClusterPhase string

// Constants representing different phases of the TensorFusionCluster resource.
const (
	TensorFusionClusterInitializing TensorFusionClusterPhase = "Initializing"
	TensorFusionClusterRunning      TensorFusionClusterPhase = "Running"
	TensorFusionClusterUpdating     TensorFusionClusterPhase = "Updating"
	TensorFusionClusterDestroying   TensorFusionClusterPhase = "Destroying"
)

type EnrollConfig struct {
	APIEndpoint string `json:"apiEndpoint,omitempty"` // API endpoint for enrollment.
	EnrollKey   struct {
		Data      string `json:"data,omitempty"` // Enrollment key data.
		SecretRef struct {
			Name      string `json:"name,omitempty"`      // Name of the secret reference.
			Namespace string `json:"namespace,omitempty"` // Namespace of the secret reference.
		} `json:"secretRef,omitempty"`
	} `json:"enrollKey,omitempty"`
}

// GPUPool represents a GPU pool configuration.
type GPUPoolDefinition struct {
	Name            string      `json:"name,omitempty"` // Name of the GPU pool.
	Spec            GPUPoolSpec `json:"spec,omitempty"`
	SpecTemplateURL string      `json:"specTemplateUrl,omitempty"`
}

// ComputingVendorConfig represents the configuration for the computing vendor.
type ComputingVendorConfig struct {
	Name                  string `json:"name,omitempty"`                  // Name of the computing vendor.
	Type                  string `json:"type,omitempty"`                  // Type of the computing vendor (e.g., aws, lambdalabs, gcp, azure, together.ai).
	AuthType              string `json:"authType,omitempty"`              // Authentication type (e.g., accessKey, serviceAccount).
	Enable                bool   `json:"enable,omitempty"`                // Enable or disable the computing vendor.
	GPUNodeControllerType string `json:"gpuNodeControllerType,omitempty"` // Type of GPU node controller (e.g., asg, karpenter, native).
	Params                struct {
		Region    string `json:"region,omitempty"`    // Region for the computing vendor.
		AccessKey string `json:"accessKey,omitempty"` // Access key for the computing vendor.
		SecretKey string `json:"secretKey,omitempty"` // Secret key for the computing vendor.
		IAMRole   string `json:"iamRole,omitempty"`   // IAM role for the computing vendor like AWS
	} `json:"params,omitempty"`
}

// StorageVendorConfig represents the configuration for the storage vendor.
type StorageVendorConfig struct {
	Mode                         string   `json:"mode,omitempty"`                         // Mode of the storage vendor (e.g., cloudnative-pg, timescale-db, RDS for PG).
	Image                        string   `json:"image,omitempty"`                        // Image for the storage vendor (default to timescale).
	InstallCloudNativePGOperator bool     `json:"installCloudNativePGOperator,omitempty"` // Whether to install CloudNative-PG operator.
	StorageClass                 string   `json:"storageClass,omitempty"`                 // Storage class for the storage vendor.
	PGExtensions                 []string `json:"pgExtensions,omitempty"`                 // List of PostgreSQL extensions to install.
	PGClusterTemplate            struct{} `json:"pgClusterTemplate,omitempty"`            // Extra spec for the PostgreSQL cluster template.
}

// DataPipelinesConfig represents the configuration for data pipelines.
type DataPipelinesConfig struct {
	Resources struct {
		SyncToCloud bool          `json:"syncToCloud,omitempty"` // Whether to sync resources to the cloud.
		SyncPeriod  time.Duration `json:"syncPeriod,omitempty"`  // Period for syncing resources.
	} `json:"resources,omitempty"`
	Timeseries struct {
		AggregationPeriods       []string          `json:"aggregationPeriods,omitempty"`       // List of aggregation periods.
		RawDataRetention         string            `json:"rawDataRetention,omitempty"`         // Retention period for raw data.
		AggregationDataRetention string            `json:"aggregationDataRetention,omitempty"` // Retention period for aggregated data.
		RemoteWrite              RemoteWriteConfig `json:"remoteWrite,omitempty"`              // Configuration for remote write.
	} `json:"timeseries,omitempty"`
}

// RemoteWriteConfig represents the configuration for remote write.
type RemoteWriteConfig struct {
	Connection struct {
		Type string `json:"type,omitempty"` // Type of the connection (e.g., datadog).
		URL  string `json:"url,omitempty"`  // URL of the connection.
	} `json:"connection,omitempty"`
	Metrics []string `json:"metrics,omitempty"` // List of metrics to remote write.
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
