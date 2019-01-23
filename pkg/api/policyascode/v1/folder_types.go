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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Folder{}, &FolderList{})
}

// FolderSpec defines the desired state of Folder
type FolderSpec struct {
	DisplayName string                 `json:"displayName"`
	ID          int64                  `json:"id,omitempty"`
	ParentRef   corev1.ObjectReference `json:"parentRef,omitempty"`
}

// FolderStatus defines the observed state of Folder
type FolderStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
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

// TFResourceConfig converts the Folder's Spec struct into terraform config string.
// It implements the terraform.Resource interface.
func (f *Folder) TFResourceConfig(ctx context.Context, c Client, tfState ResourceState) (string, error) {
	var parent string
	pKind := f.Spec.ParentRef.Kind
	switch pKind {
	case OrganizationKind, FolderKind:
		id, err := ResourceID(ctx, c, f.Spec.ParentRef.Kind, f.Spec.ParentRef.Name, EmptyNamespace)
		if err != nil {
			return "", err
		}
		if pKind == OrganizationKind {
			parent = fmt.Sprintf("organizations/%s", id)
		} else {
			parent = fmt.Sprintf("folders/%s", id)
		}
	case "":
		// Terraform requires the parent field to present, get the parent from Terraform local state.
		p, ok := tfState["parent"]
		if !ok || p == "" {
			return "", fmt.Errorf("missing parent from terraform state")
		}
		parent = p
	default:
		return "", fmt.Errorf("invalid parent reference kind: %v", pKind)
	}

	return fmt.Sprintf(`resource "google_folder" "bespin_folder" {
display_name = "%s"
parent = "%s"
}`, f.Spec.DisplayName, parent), nil
}

// TFImportConfig returns an empty terraform Folder resource block used for terraform import.
// It implements the terraform.Resource interface.
func (f *Folder) TFImportConfig() string {
	return `resource "google_folder" "bespin_folder" {}`
}

// TFResourceAddr returns the address of this Folder resource in terraform config.
// It implements the terraform.Resource interface.
func (f *Folder) TFResourceAddr() string {
	return `google_folder.bespin_folder`
}

// ID returns the Folder ID from GCP.
// It implements the terraform.Resource interface.
func (f *Folder) ID() string {
	if f.Spec.ID != 0 {
		return fmt.Sprintf("%v", f.Spec.ID)
	}
	return ""
}

// ReferenceID implements the terraform.Resource interface.
// For Folder objects, it returns empty string.
func (f *Folder) ReferenceID(ctx context.Context, c Client) (string, error) {
	return "", nil
}
