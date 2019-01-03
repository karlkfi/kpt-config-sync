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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}

// OrganizationSpec defines the desired state of Organization
type OrganizationSpec struct {
	ID int64 `json:"id"`
}

// OrganizationStatus defines the observed state of Organization
type OrganizationStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
}

// Organization is the Schema for the organizations API
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// OrganizationList contains a list of Organization
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

// TFResourceConfig converts the Organization's Spec struct into terraform config string.
// Organizations are different from other GCP resources that they are not allowed to be created,
// updated (OrgPolicy can be attached, but not update the Organization itself), deleted. It's
// READONLY in bespin world, and in Terraform there is only a "data" config (no "resource")
// for an Organization.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (o *Organization) TFResourceConfig(ctx context.Context, c Client) (string, error) {
	ID := o.ID()
	if ID == "" {
		return "", fmt.Errorf("missing Organization ID")
	}
	return fmt.Sprintf(`data "google_organization" "bespin_organization" {
organization = "organizations/%v"
}`, ID), nil
}

// TFImportConfig returns an empty terraform organization resource block used for terraform import.
// The string is NOT applicable for google_organization because there doesn't exist a "resource" for
// google_organization and trying to import an organization in Terraform is not supported in Terraform.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (o *Organization) TFImportConfig() string {
	return ""
}

// TFResourceAddr returns the address of this Organization resource in terraform config.
// The string is NOT applicable for google_organization because there doesn't exist a "resource" for
// google_organization and trying to import an organization in Terraform is not supported in Terraform.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (o *Organization) TFResourceAddr() string {
	return ""
}

// ID returns the Organization ID from GCP.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (o *Organization) ID() string {
	if o.Spec.ID == 0 {
		return ""
	}
	return fmt.Sprintf("%v", o.Spec.ID)
}
