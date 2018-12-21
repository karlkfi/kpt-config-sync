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
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (i *IAMPolicy) TFResourceConfig(ctx context.Context, c Client) (string, error) {
	var tfs []string
	resName := types.NamespacedName{Name: i.Spec.ResourceReference.Name}
	switch i.Spec.ResourceReference.Kind {
	case OrganizationKind:
		org := &Organization{}
		if err := c.Get(ctx, resName, org); err != nil {
			return "", errors.Wrapf(err, "failed to get reference resource Organization instance: %v", resName)
		}
		ID := org.ID()
		if ID == "" {
			return "", fmt.Errorf("missing resource reference Organization ID: %v", resName)
		}
		tfs = append(tfs, `resource "google_organization_iam_policy" "bespin_organization_iam_policy" {`)
		tfs = append(tfs, fmt.Sprintf(`org_id = "organizations/%s"`, ID))
	case FolderKind:
		folder := &Folder{}
		if err := c.Get(ctx, resName, folder); err != nil {
			return "", errors.Wrapf(err, "failed to get reference resource Folder instance: %v", resName)
		}
		ID := folder.ID()
		if ID == "" {
			return "", fmt.Errorf("missing resource reference Folder ID: %v", resName)
		}
		tfs = append(tfs, `resource "google_folder_iam_policy" "bespin_folder_iam_policy" {`)
		tfs = append(tfs, fmt.Sprintf(`folder = "folders/%s"`, ID))
	case ProjectKind:
		resName = types.NamespacedName{Namespace: i.Namespace, Name: i.Spec.ResourceReference.Name}
		project := &Project{}
		if err := c.Get(ctx, resName, project); err != nil {
			return "", errors.Wrapf(err, "failed to get resource reference Project instance: %v", resName)
		}
		ID := project.ID()
		if ID == "" {
			return "", fmt.Errorf("missing resource reference Project ID: %v", resName)
		}
		tfs = append(tfs, `resource "google_project_iam_policy" "bespin_project_iam_policy" {`)
		tfs = append(tfs, fmt.Sprintf(`project = "%s"`, ID))
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

// TFImportConfig returns an empty terraform IAMPolicy resource block used for terraform import.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (i *IAMPolicy) TFImportConfig() string {
	return `resource "google_project_iam_policy" "project_iam_policy" {}`
}

// TFResourceAddr returns the address of this IAMPolicy resource in Terraform config.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (i *IAMPolicy) TFResourceAddr() string {
	return `google_project_iam_policy.project_iam_policy`
}

// ID doesn't apply to IAMPolicy.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (i *IAMPolicy) ID() string {
	return ""
}
