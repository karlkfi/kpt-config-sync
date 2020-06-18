/*


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

// +kubebuilder:object:root=true

// RootSyncSpec defines the desired state of RootSync
type RootSyncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	// Git contains configuration specific to importing policies from a Git repo.
	// +optional
	Git `json:"git,omitempty"`
}

// +kubebuilder:object:root=true

// RootSyncStatus defines the observed state of RootSync
type RootSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// RootSync is the Schema for the rootsyncs API
type RootSync struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec RootSyncSpec `json:"spec,omitempty"`
	// +optional
	Status RootSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RootSyncList contains a list of RootSync
type RootSyncList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootSync `json:"items"`
}
