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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	DisplayName string                 `json:"displayName"`
	ID          string                 `json:"id"`
	ParentRef   corev1.ObjectReference `json:"parentRef,omitempty"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
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

// TFResourceConfig converts the Project's Spec struct into terraform config string.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (p *Project) TFResourceConfig(ctx context.Context, c Client, tfState map[string]string) (string, error) {
	var parent string
	resName := types.NamespacedName{Name: p.Spec.ParentRef.Name}
	switch p.Spec.ParentRef.Kind {
	case OrganizationKind:
		org := &Organization{}
		if err := c.Get(ctx, resName, org); err != nil {
			return "", errors.Wrapf(err, "failed to get parent Organization instance: %v", resName)
		}
		ID := org.ID()
		if ID == "" {
			return "", fmt.Errorf("missing parent Organization ID: %v", resName)
		}
		parent = fmt.Sprintf(`org_id = "%s"`, ID)
	case FolderKind:
		folder := &Folder{}
		if err := c.Get(ctx, resName, folder); err != nil {
			return "", errors.Wrapf(err, "failed to get parent Folder instance: %v", resName)
		}
		ID := folder.ID()
		if ID == "" {
			return "", fmt.Errorf("missing parent Folder ID: %v", resName)
		}
		parent = fmt.Sprintf(`folder_id = "%s"`, ID)
	case "":
		if p.Spec.ParentRef.Name != "" {
			return "", fmt.Errorf("invalid parent reference name when parent reference kind is missing: %v", p.Spec.ParentRef.Name)
		}
		// No parent reference.
	default:
		return "", fmt.Errorf("invalid parent reference kind: %v", p.Spec.ParentRef.Kind)
	}

	return fmt.Sprintf(`resource "google_project" "bespin_project" {
name = "%s"
project_id = "%s"
%s
}`, p.Spec.DisplayName, p.ID(), parent), nil
}

// TFImportConfig returns an empty terraform project resource block used for terraform import.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (p *Project) TFImportConfig() string {
	return `resource "google_project" "bespin_project" {}`
}

// TFResourceAddr returns the address of this project resource in terraform config.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (p *Project) TFResourceAddr() string {
	return `google_project.bespin_project`
}

// ID returns the project ID from GCP.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (p *Project) ID() string {
	return p.Spec.ID
}
