package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FolderSpec defines the desired state of Folder
type FolderSpec struct {
	DisplayName string                 `json:"displayName"`
	ID          int64                  `json:"id,omitempty"`
	ParentRef   corev1.ObjectReference `json:"parentRef,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// Folder is the Schema for the folders API
// +k8s:openapi-gen=true
type Folder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FolderSpec `json:"spec"`
	Status Status     `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FolderList contains a list of Folder
type FolderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Folder `json:"items"`
}

// Conditions returns the list of conditions of resource status.
// It implements the resource.GenericObject interface.
func (f *Folder) Conditions() []Condition {
	return f.Status.Conditions
}
