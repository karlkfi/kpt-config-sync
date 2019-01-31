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
	SchemeBuilder.Register(&IAMPolicy{}, &IAMPolicyList{})
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMPolicy is the Schema for the iampolicies API
// +k8s:openapi-gen=true
type IAMPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMPolicySpec   `json:"spec"`
	Status IAMPolicyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMPolicyList contains a list of IAMPolicy
type IAMPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IAMPolicy `json:"items"`
}

// TFResourceConfig converts the IAMPolicy's Spec struct into terraform config string.
// It implements the terraform.Resource interface.
func (i *IAMPolicy) TFResourceConfig(ctx context.Context, c Client, tfState ResourceState) (string, error) {
	if i.Spec.ResourceRef.Kind != ProjectKind {
		return "", fmt.Errorf("invalid resource reference kind for IAMPolicy: %v", i.Spec.ResourceRef.Kind)
	}
	id, err := i.ReferenceID(ctx, c)
	if err != nil {
		return "", err
	}
	policyData := "${data.google_iam_policy.admin.policy_data}"
	if len(i.Spec.Bindings) == 0 {
		policyData = "{}"
	}
	tfPolicy := fmt.Sprintf(`resource "google_project_iam_policy" "bespin_project_iam_policy" {
project = "%s"
policy_data = "%s"
}
`, id, policyData)
	return tfPolicy + i.Spec.TFBindingsConfig(), nil
}

// TFImportConfig returns a terraform IAMPolicy resource block used for terraform import.
// It implements the terraform.Resource interface.
func (i *IAMPolicy) TFImportConfig() string {
	return `resource "google_project_iam_policy" "bespin_project_iam_policy" {}`
}

// TFResourceAddr returns the address of this IAMPolicy resource in Terraform config.
// It implements the terraform.Resource interface.
func (i *IAMPolicy) TFResourceAddr() string {
	return `google_project_iam_policy.bespin_project_iam_policy`
}

// ID returns the reference resource ID.
// TODO(b/122925391): fetch resource reference ID from api server.
// It implements the terraform.Resource interface.
func (i *IAMPolicy) ID() string {
	return ""
}

// ReferenceID implements the terraform.Resource interface.
// It returns the Project ID on GCP where the IAMPolicy points to.
func (i *IAMPolicy) ReferenceID(ctx context.Context, c Client) (string, error) {
	id, err := ResourceID(ctx, c, i.Spec.ResourceRef.Kind, i.Spec.ResourceRef.Name, i.Namespace)
	if err != nil {
		return "", err
	}
	return id, nil
}
