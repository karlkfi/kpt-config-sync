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

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
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

var folderSync = &v1alpha1.Sync{
	Spec: v1alpha1.SyncSpec{
		Groups: []v1alpha1.SyncGroup{
			{
				Group: "bespin.dev",
				Kinds: []v1alpha1.SyncKind{
					{
						Kind: "Folder",
						Versions: []v1alpha1.SyncVersion{
							{
								Version: "v1",
							},
						},
					},
				},
			},
		},
	},
}

func init() {
	SchemeBuilder.Register(&Folder{}, &FolderList{})
}

// GetTFResourceConfig converts the Folder's Spec struct into terraform config string.
func (f *Folder) GetTFResourceConfig() (string, error) {
	switch f.Spec.ParentReference.Kind {
	case "Organization", "Folder":
		break
	default:
		return "", fmt.Errorf("invalid parent reference kind: %v", f.Spec.ParentReference.Kind)
	}

	return fmt.Sprintf(`resource "google_folder" "bespin_folder" {
display_name = "%s"
parent = "%s"
}`, f.Spec.DisplayName, f.Spec.ParentReference.Name), nil
}

// GetTFImportConfig returns an empty terraform project resource block used for terraform import.
func (f *Folder) GetTFImportConfig() string {
	return `resource "google_folder" "bespin_folder" {}`
}

// GetTFResourceAddr returns the address of this project resource in terraform config.
func (f *Folder) GetTFResourceAddr() string {
	return `google_folder.bespin_folder`
}

// GetID returns the project ID from underlying provider (e.g. GCP).
func (f *Folder) GetID() string {
	return fmt.Sprintf("%v", f.Spec.ID)
}
