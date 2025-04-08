/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceConfig defines the configuration for a microservice
type ServiceConfig struct {
	// Image is the container image for the microservice
	Image string `json:"image"`

	// Port is the container port for the microservice
	Port int32 `json:"port"`

	// Replicas is the number of desired replicas
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources defines the resource requirements for the microservice
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// Environment variables for the microservice
	// +optional
	Env []EnvVar `json:"env,omitempty"`
}

// ResourceRequirements defines the compute resources required
type ResourceRequirements struct {
	// CPU resource requirements
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory resource requirements
	// +optional
	Memory string `json:"memory,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	// Name of the environment variable
	Name string `json:"name"`

	// Value of the environment variable
	Value string `json:"value"`
}

// GitOpsConfig defines the GitOps configuration
type GitOpsConfig struct {
	// Repository URL containing the service configuration
	RepoURL string `json:"repoURL"`

	// Branch to track for changes
	Branch string `json:"branch"`

	// Path within the repository for service configuration
	Path string `json:"path"`

	// Automated sync configuration
	// +optional
	AutoSync *bool `json:"autoSync,omitempty"`
}

// IstioConfig defines the service mesh configuration
type IstioConfig struct {
	// Enable Istio sidecar injection
	Enabled bool `json:"enabled"`

	// VirtualService configuration
	// +optional
	VirtualService *VirtualServiceConfig `json:"virtualService,omitempty"`

	// DestinationRule configuration
	// +optional
	DestinationRule *DestinationRuleConfig `json:"destinationRule,omitempty"`
}

// VirtualServiceConfig defines the Istio VirtualService settings
type VirtualServiceConfig struct {
	// Hosts to which traffic is being sent
	Hosts []string `json:"hosts"`

	// Gateways list the names of gateways and sidecars
	// +optional
	Gateways []string `json:"gateways,omitempty"`
}

// DestinationRuleConfig defines the Istio DestinationRule settings
type DestinationRuleConfig struct {
	// Traffic policy settings
	// +optional
	TrafficPolicy *TrafficPolicy `json:"trafficPolicy,omitempty"`
}

// TrafficPolicy defines the traffic management policy
type TrafficPolicy struct {
	// LoadBalancer settings
	// +optional
	LoadBalancer *LoadBalancerSettings `json:"loadBalancer,omitempty"`
}

// LoadBalancerSettings defines load balancing settings
type LoadBalancerSettings struct {
	// Simple load balancing
	Simple string `json:"simple,omitempty"`
}

// MicroserviceConfigSpec defines the desired state of MicroserviceConfig
type MicroserviceConfigSpec struct {
	// Service configuration
	Service ServiceConfig `json:"service"`

	// GitOps configuration for the service
	GitOps GitOpsConfig `json:"gitops"`

	// Istio service mesh configuration
	// +optional
	Istio *IstioConfig `json:"istio,omitempty"`
}

// MicroserviceConfigStatus defines the observed state of MicroserviceConfig
type MicroserviceConfigStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of configuration rollout
	// +optional
	Phase string `json:"phase,omitempty"`

	// GitOps sync status
	// +optional
	GitOpsSyncStatus string `json:"gitOpsSyncStatus,omitempty"`

	// Last time the configuration was updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Sync Status",type="string",JSONPath=".status.gitOpsSyncStatus"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// MicroserviceConfig is the Schema for the microserviceconfigs API
type MicroserviceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicroserviceConfigSpec   `json:"spec,omitempty"`
	Status MicroserviceConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MicroserviceConfigList contains a list of MicroserviceConfig
type MicroserviceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicroserviceConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicroserviceConfig{}, &MicroserviceConfigList{})
}
