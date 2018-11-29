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

// TFString convert the ProjectSpec struct into terraform appliable string
func (ps *ProjectSpec) TFString() (string, error) {
	parentReference := ""
	if ps.ParentReference.Kind == "Organization" {
		parentReference = fmt.Sprintf(`org_id = "%s"`, ps.ParentReference.Name)
	} else if ps.ParentReference.Kind == "Folder" {
		parentReference = fmt.Sprintf(`folder_id = "%s"`, ps.ParentReference.Name)
	} else {
		return "", fmt.Errorf("error parent kind: %v", ps.ParentReference.Kind)
	}

	tfs := fmt.Sprintf(
		`resource "google_project" "my_project" {
		   name = "%s"
		   project_id = "%s"
		   %s
		}`, ps.Name, ps.ID, parentReference)
	return tfs, nil
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

var projectSync = &v1alpha1.Sync{
	Spec: v1alpha1.SyncSpec{
		Groups: []v1alpha1.SyncGroup{
			{
				Group: "bespin.dev",
				Kinds: []v1alpha1.SyncKind{
					{
						Kind: "Project",
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
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
