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

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// +kubebuilder:validation:Pattern=[a-zA-Z][\w!"\- ]{3,27}
	Name string `json:"name"`
	// +kubebuilder:validation:Pattern=^[a-z][a-z\d\-]{5,29}$
	ID              string          `json:"id"`
	ParentReference ParentReference `json:"parentReference,omitempty"`
	Labels          ProjectLabels   `json:"labels,omitempty"`
	ImportDetails   ImportDetails   `json:"importDetails"`
}

// GetTFResourceConfig converts the Project's Spec struct into terraform config string.
func (p *Project) GetTFResourceConfig() (string, error) {
	parentReference := ""
	if p.Spec.ParentReference.Kind == OrganizationKind {
		parentReference = fmt.Sprintf(`org_id = "%s"`, p.Spec.ParentReference.Name)
	} else if p.Spec.ParentReference.Kind == FolderKind {
		parentReference = fmt.Sprintf(`folder_id = "%s"`, p.Spec.ParentReference.Name)
	} else {
		return "", fmt.Errorf("invalid parent reference kind: %v", p.Spec.ParentReference.Kind)
	}

	return fmt.Sprintf(`resource "google_project" "bespin_project" {
name = "%s"
project_id = "%s"
%s
}`, p.Spec.Name, p.Spec.ID, parentReference), nil
}

// GetTFImportConfig returns an empty terraform project resource block used for terraform import.
func (p *Project) GetTFImportConfig() string {
	return `resource "google_project" "bespin_project" {}`
}

// GetTFResourceAddr returns the address of this project resource in terraform config.
func (p *Project) GetTFResourceAddr() string {
	return `google_project.bespin_project`
}

// GetID returns the project ID from underlying provider (e.g. GCP).
func (p *Project) GetID() string {
	return p.Spec.ID
}

// ProjectLabels defines a label dictionary for CRD resources
// https://swagger.io/docs/specification/data-models/dictionaries/
type ProjectLabels struct {
	AdditionalProperties string `json:"additionalProperties"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	SyncDetails SyncDetails `json:"syncDetails,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project is the Schema for the projects API
// +k8s:openapi-gen=true
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
