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
	"strings"

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

// GetTFResourceConfig converts the IAMPolicy's Spec struct into terraform config string.
func (i *IAMPolicy) GetTFResourceConfig() (string, error) {
	var tfs []string
	switch i.Spec.ResourceReference.Kind {
	case OrganizationKind:
		tfs = append(tfs, `resource "google_organization_iam_policy" "organization_iam_policy" {`)
		tfs = append(tfs, fmt.Sprintf(`org_id = "%s"`, i.Spec.ResourceReference.Name))
	case FolderKind:
		tfs = append(tfs, `resource "google_folder_iam_policy" "folder_iam_policy" {`)
		tfs = append(tfs, fmt.Sprintf(`folder = "%s"`, i.Spec.ResourceReference.Name))
	case ProjectKind:
		tfs = append(tfs, `resource "google_project_iam_policy" "project_iam_policy" {`)
		tfs = append(tfs, fmt.Sprintf(`project = "%s"`, i.Spec.ResourceReference.Name))
	default:
		return "", fmt.Errorf("invalid resource reference kind: %v", i.Spec.ResourceReference.Kind)
	}
	tfs = append(tfs, `policy_data = "${data.google_iam_policy.admin.policy_data}"`)
	tfs = append(tfs, `}`)
	// IAM policy data.
	// Example:
	// data "google_iam_policy" "admin" {
	//   binding {
	//    role = "roles/compute.instanceAdmin"

	//    members = [
	//      "serviceAccount:your-custom-sa@your-project.iam.gserviceaccount.com",
	//    ]
	//  }
	//   binding {
	//     role = "roles/storage.objectViewer"

	//     members = [
	//       "user:jane@example.com",
	//     ]
	//   }
	// }
	tfs = append(tfs, (`data "google_iam_policy" "admin" {`))
	for _, b := range i.Spec.Bindings {
		tfs = append(tfs, `binding {`)
		tfs = append(tfs, fmt.Sprintf(`role = "%s"`, b.Role))
		tfs = append(tfs, `members = [`)
		for _, m := range b.Members {
			tfs = append(tfs, fmt.Sprintf(`"%s",`, m))
		}
		tfs = append(tfs, `]}`)
	}
	tfs = append(tfs, `}`)
	return strings.Join(tfs, "\n"), nil
}

// GetTFImportConfig returns an empty terraform project resource block used for terraform import.
func (i *IAMPolicy) GetTFImportConfig() string {
	return `resource "google_project_iam_policy" "project_iam_policy" {}`
}

// GetTFResourceAddr returns the address of this project resource in terraform config.
func (i *IAMPolicy) GetTFResourceAddr() string {
	return `google_project_iam_policy.project_iam_policy`
}

// GetID returns the project ID from underlying provider (e.g. GCP).
func (i *IAMPolicy) GetID() string {
	return i.Spec.ResourceReference.Name
}
