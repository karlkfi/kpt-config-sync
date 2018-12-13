/*
Copyright 2018 Google LLC.

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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FolderSpec defines the desired state of Folder
type FolderSpec struct {
	// +kubebuilder:validation:Pattern=[a-zA-Z\d][\w_ \-]{3,27}[\w\d]?
	DisplayName string `json:"displayName"`
	// +kubebuilder:validation:Minimum=1
	ID              int             `json:"id,omitempty"`
	ParentReference ParentReference `json:"parentReference,omitempty"`
	ImportDetails   ImportDetails   `json:"importDetails"`
}

// FolderStatus defines the observed state of Folder
type FolderStatus struct {
	ID          int         `json:"id,omitempty"`
	SyncDetails SyncDetails `json:"syncDetails,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// Folder is the Schema for the folders API
// +k8s:openapi-gen=true
type Folder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FolderSpec   `json:"spec"`
	Status FolderStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FolderList contains a list of Folder
type FolderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Folder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Folder{}, &FolderList{})
}

// GetTFResourceConfig converts the Folder's Spec struct into terraform config string.
func (f *Folder) GetTFResourceConfig() (string, error) {
	var parent string
	annotations := f.GetAnnotations()
	switch f.Spec.ParentReference.Kind {
	case OrganizationKind:
		orgID, ok := annotations[ParentOrganizationIDKey]
		if !ok {
			return "", fmt.Errorf("parent Organization ID not found in annotations: %v", ParentOrganizationIDKey)
		}
		parent = fmt.Sprintf("organizations/%s", orgID)
	case FolderKind:
		folderID, ok := annotations[ParentFolderIDKey]
		if !ok {
			return "", fmt.Errorf("parent Folder ID not found in annotations: %v", ParentFolderIDKey)
		}
		parent = fmt.Sprintf("folders/%s", folderID)
	default:
		return "", fmt.Errorf("invalid parent reference kind: %v", f.Spec.ParentReference.Kind)
	}

	return fmt.Sprintf(`resource "google_folder" "bespin_folder" {
display_name = "%s"
parent = "%s"
}`, f.Spec.DisplayName, parent), nil
}

// GetTFImportConfig returns an empty terraform Folder resource block used for terraform import.
func (f *Folder) GetTFImportConfig() string {
	return `resource "google_folder" "bespin_folder" {}`
}

// GetTFResourceAddr returns the address of this Folder resource in terraform config.
func (f *Folder) GetTFResourceAddr() string {
	return `google_folder.bespin_folder`
}

// GetID returns the Folder ID from GCP. It first looks at Status.ID, and use that
// if present, if not it uses Spec.ID. When there is no ID present, Spec.ID will
// be 0 and returned.
func (f *Folder) GetID() string {
	if f.Status.ID != 0 {
		return fmt.Sprintf("%v", f.Status.ID)
	}
	return fmt.Sprintf("%v", f.Spec.ID)
}

// GetParentReference returns the Folder ParentRefernce.
func (f *Folder) GetParentReference() ParentReference {
	return f.Spec.ParentReference
}

// Validate does sanity check on the Folder resource, and returns error if any
// inconsistency found.
func (f *Folder) Validate() error {
	// Invalid if Spec.ID and Status.ID both present but not equal.
	if f.Spec.ID != 0 && f.Status.ID != 0 && f.Spec.ID != f.Status.ID {
		return fmt.Errorf("inconsistent Foder Spec ID (%v) and Folder Status ID (%v)", f.Spec.ID, f.Status.ID)
	}
	return nil
}
