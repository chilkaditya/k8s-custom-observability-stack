/*
Copyright 2026.

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

// CustomObservabilityStackSpec defines the desired state of CustomObservabilityStack.
type CustomObservabilityStackSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of CustomObservabilityStack. Edit customobservabilitystack_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	TargetDeployment string `json:"targetDeployment"`
	TargetNamespace  string `json:"targetNamespace"`
	CPUThreshold     int32  `json:"cpuThreshold"`
	MemoryThreshold  int32  `json:"memoryThreshold"`
	EnableDashboard  bool   `json:"enableDashboard"`
}

// CustomObservabilityStackStatus defines the observed state of CustomObservabilityStack.
type CustomObservabilityStackStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Phase              string `json:"phase,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	LastReconcileTime  string `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CustomObservabilityStack is the Schema for the customobservabilitystacks API.
type CustomObservabilityStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomObservabilityStackSpec   `json:"spec,omitempty"`
	Status CustomObservabilityStackStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CustomObservabilityStackList contains a list of CustomObservabilityStack.
type CustomObservabilityStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomObservabilityStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CustomObservabilityStack{}, &CustomObservabilityStackList{})
}
