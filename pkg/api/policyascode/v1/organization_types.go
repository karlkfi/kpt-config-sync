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

// OrganizationSpec defines the desired state of Organization
type OrganizationSpec struct {
	// +kubebuilder:validation:Minimum=1
	ID            int           `json:"id"`
	ImportDetails ImportDetails `json:"importDetails"`
}

// OrganizationStatus defines the observed state of Organization
type OrganizationStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// Organization is the Schema for the organizations API
// +k8s:openapi-gen=true
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OrganizationList contains a list of Organization
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}

// GetTFResourceConfig converts the Organization's Spec struct into terraform config string.
// Organizations are different from other GCP resources that they are not allowed to be created,
// updated (OrgPolicy can be attached, but not update the Organization itself), deleted. It's
// READONLY in bespin world, and in Terraform there is only a "data" config (no "resource")
// for an Organization.
func (o *Organization) GetTFResourceConfig() (string, error) {
	if o.Spec.ID == 0 {
		return "", fmt.Errorf("invalid organization ID: 0")
	}
	return fmt.Sprintf(`data "google_organization" "bespin_organization" {
organization = "organizations/%v"
}`, o.Spec.ID), nil
}

// GetTFImportConfig returns an empty terraform organization resource block used for terraform import.
// The string is NOT applicable for google_organization because there doesn't exist a "resource" for
// google_organization and trying to import an organization in Terraform is not supported in Terraform.
// Making this function return empty string just to make the Interface happy.
func (o *Organization) GetTFImportConfig() string {
	return ""
}

// GetTFResourceAddr returns the address of this Organization resource in terraform config.
// The string is NOT applicable for google_organization because there doesn't exist a "resource" for
// google_organization and trying to import an organization in Terraform is not supported in Terraform.
// Making this function return empty string just to make the Interface happy.
func (o *Organization) GetTFResourceAddr() string {
	return ""
}

// GetID returns the Organization ID from GCP.
func (o *Organization) GetID() string {
	return fmt.Sprintf("%v", o.Spec.ID)
}
