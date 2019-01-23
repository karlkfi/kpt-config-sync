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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterIAMPolicy is the Schema for the clusteriampolicies API
// +k8s:openapi-gen=true
type ClusterIAMPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMPolicySpec   `json:"spec,omitempty"`
	Status IAMPolicyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterIAMPolicyList contains a list of ClusterIAMPolicy
type ClusterIAMPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterIAMPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterIAMPolicy{}, &ClusterIAMPolicyList{})
}

// TFResourceConfig converts the ClusterIAMPolicy's Spec struct into terraform config string.
// It implements the terraform.Resource interface.
func (i *ClusterIAMPolicy) TFResourceConfig(ctx context.Context, c Client, tfState ResourceState) (string, error) {
	var tfs string
	refKind := i.Spec.ResourceRef.Kind
	id, err := i.ReferenceID(ctx, c)
	if err != nil {
		return "", err
	}
	switch refKind {
	case OrganizationKind:
		tfs = fmt.Sprintf(`resource "google_organization_iam_policy" "bespin_organization_iam_policy" {
org_id = "organizations/%s"
`, id)
	case FolderKind:
		tfs = fmt.Sprintf(`resource "google_folder_iam_policy" "bespin_folder_iam_policy" {
folder = "folders/%s"
`, id)
	default:
		return "", fmt.Errorf("invalid resource reference kind for ClusterIAMPolicy: %v", refKind)
	}
	// TODO(b/122963799): format Terraform config string using https://github.com/hashicorp/hcl/blob/master/hcl/printer/printer.go
	tfs = tfs + `policy_data = "${data.google_iam_policy.admin.policy_data}"
}
`
	return tfs + i.Spec.TFBindingsConfig(), nil
}

// TFImportConfig returns a terraform ClusterIAMPolicy resource block used for terraform import.
// It implements the terraform.Resource interface.
func (i *ClusterIAMPolicy) TFImportConfig() string {
	switch i.Spec.ResourceRef.Kind {
	case OrganizationKind:
		return `resource "google_organization_iam_policy" "bespin_organization_iam_policy" {}`
	case FolderKind:
		return `resource "google_folder_iam_policy" "bespin_folder_iam_policy" {}`
	default:
		return ""
	}
}

// TFResourceAddr returns the address of this ClusterIAMPolicy resource in Terraform config.
// It implements the terraform.Resource interface.
func (i *ClusterIAMPolicy) TFResourceAddr() string {
	switch i.Spec.ResourceRef.Kind {
	case OrganizationKind:
		return "google_organization_iam_policy.bespin_organization_iam_policy"
	case FolderKind:
		return "google_folder_iam_policy.bespin_folder_iam_policy"
	default:
		return ""
	}
}

// ID returns the reference resource ID.
// TODO(b/122925391): fetch resource reference ID from api server.
// It implements the terraform.Resource interface.
func (i *ClusterIAMPolicy) ID() string {
	return ""
}

// ReferenceID implements the terraform.Resource interface.
func (i *ClusterIAMPolicy) ReferenceID(ctx context.Context, c Client) (string, error) {
	id, err := ResourceID(ctx, c, i.Spec.ResourceRef.Kind, i.Spec.ResourceRef.Name, EmptyNamespace)
	if err != nil {
		return "", err
	}
	return id, nil
}
