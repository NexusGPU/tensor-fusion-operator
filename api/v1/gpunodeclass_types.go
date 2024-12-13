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

// GPUNodeClassSpec defines the desired state of GPUNodeClass.
type GPUNodeClassSpec struct {
	OSImageFamily string `json:"osImageFamily,omitempty"` // The AMI family to use

	OSImageSelectorTerms []NodeClassOSImageSelectorTerms `json:"osImageSelectorTerms,omitempty"`

	BlockDeviceMappings []NodeClassBlockDeviceMappings `json:"blockDeviceMappings,omitempty"` // Block device mappings for the instance

	InstanceProfile string `json:"instanceProfile,omitempty"` // The instance profile to use

	MetadataOptions NodeClassMetadataOptions `json:"metadataOptions,omitempty"`

	SecurityGroupSelectorTerms []NodeClassItemIDSelectorTerms `json:"securityGroupSelectorTerms,omitempty"`

	SubnetSelectorTerms []NodeClassItemIDSelectorTerms `json:"subnetSelectorTerms,omitempty"` // Terms to select subnets

	Tags map[string]string `json:"tags,omitempty"` // Tags associated with the resource

	UserData string `json:"userData,omitempty"` // User data script for the instance
}

type NodeClassItemIDSelectorTerms struct {
	ID string `json:"id,omitempty"` // The ID of the security group
}

type NodeClassMetadataOptions struct {
	HttpEndpoint            string `json:"httpEndpoint,omitempty"`            // Whether the HTTP metadata endpoint is enabled
	HttpProtocolIPv6        string `json:"httpProtocolIPv6,omitempty"`        // Whether IPv6 is enabled for the HTTP metadata endpoint
	HttpPutResponseHopLimit int    `json:"httpPutResponseHopLimit,omitempty"` // The hop limit for HTTP PUT responses
	HttpTokens              string `json:"httpTokens,omitempty"`              // The HTTP tokens required for metadata access
}

type NodeClassOSImageSelectorTerms struct {
	Name  string `json:"name,omitempty"`
	Owner string `json:"owner,omitempty"`
}

type NodeClassBlockDeviceMappings struct {
	DeviceName string               `json:"deviceName,omitempty"` // The device name for the block device
	Ebs        NodeClassEbsSettings `json:"ebs,omitempty"`
}

type NodeClassEbsSettings struct {
	DeleteOnTermination bool   `json:"deleteOnTermination,omitempty"` // Whether to delete the EBS volume on termination
	Encrypted           bool   `json:"encrypted,omitempty"`           // Whether the EBS volume is encrypted
	VolumeSize          string `json:"volumeSize,omitempty"`          // The size of the EBS volume
	VolumeType          string `json:"volumeType,omitempty"`          // The type of the EBS volume
}

// GPUNodeClassStatus defines the observed state of GPUNodeClass.
type GPUNodeClassStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// GPUNodeClass is the Schema for the gpunodeclasses API.
type GPUNodeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GPUNodeClassSpec   `json:"spec,omitempty"`
	Status GPUNodeClassStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GPUNodeClassList contains a list of GPUNodeClass.
type GPUNodeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GPUNodeClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GPUNodeClass{}, &GPUNodeClassList{})
}
