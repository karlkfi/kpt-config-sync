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

	"github.com/golang/glog"

	// hcl/printer is vendored and will be used in a future CL.
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OrganizationPolicy is the Schema for the organizationpolicies API
type OrganizationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationPolicySpec   `json:"spec"`
	Status OrganizationPolicyStatus `json:"status,omitempty"`
}

// Terraform configuration generation section.
// For general documentation on organization policy see:
// https://cloud.google.com/resource-manager/reference/rest/v1/organizations
// For TF/GCP provider docs see:
// https://www.terraform.io/docs/providers/google/r/google_organization_policy.html
// https://www.terraform.io/docs/providers/google/r/google_project_organization_policy.html

func tfPolicyResourceName(kind string) string {
	switch kind {
	case OrganizationKind:
		return "google_organization_policy"
	case FolderKind:
		return "google_folder_organization_policy"
	case ProjectKind:
		return "google_project_organization_policy"
	default:
		panic(fmt.Sprintf("unknown kind %s, should not happen", kind))
	}
}

func tfPolicyResourceKeyName(kind string) string {
	switch kind {
	case OrganizationKind:
		return "org_id"
	case FolderKind:
		return "folder"
	case ProjectKind:
		return "project"
	default:
		panic(fmt.Sprintf("unknown kind %s, should not happen", kind))
	}
}

// quoteList turns a list of strings into a string formatted for TF.
func quoteList(values []string) string {
	result := make([]string, len(values))
	for i, s := range values {
		result[i] = fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("[%v]", strings.Join(result, ", "))
}

// tfFormatListPolicyResource formats a OrganizationPolicyListPolicy.
//   See https://cloud.google.com/resource-manager/reference/rest/v1/Policy#ListPolicy
// for a description of how this works.
// Brief outline:
// - if AllValues is set, then AllowedValues and DisallowedValues should be empty.
// - We could in theory set both Allowed and Disallowed values on the same policy, but its not clear
//     why you would want to.  The TF docs are unclear if it allows both to be set.
// - On the other hand, the *computed* policy may have both set due to inheritance, so the gcp API might
//     give us a policy with this configuration.
// TODO(b/121396490) support suggested_values field?
func tfFormatListPolicyResource(lp OrganizationPolicyListPolicy) (string, error) {
	var result = "list_policy { "
	// Convenient format string for list_policy contents.
	var listPolicyTmpl = `%s {
	%s = %s
}
`
	// If allValues is set to either ALLOW or DENY,
	if lp.AllValues != "" {
		switch lp.AllValues {
		case "ALLOW":
			result += fmt.Sprintf(listPolicyTmpl, "allow", "all", "true")
		case "DENY":
			result += fmt.Sprintf(listPolicyTmpl, "deny", "all", "true")
		default:
			return "", errors.Errorf("Bad value %v for AllValues", lp.AllValues)
		} // TODO(lschumacher): need an err here?
	} else {
		// TODO(lschumacher): verify if the terraform provider allows both clauses to be present.
		// the docs are ambiguous on this piont.
		if len(lp.AllowedValues) > 0 {
			result += fmt.Sprintf(listPolicyTmpl, "allow", "values", quoteList(lp.AllowedValues))
		}
		if len(lp.DisallowedValues) > 0 {
			result += fmt.Sprintf(listPolicyTmpl, "deny", "values", quoteList(lp.DisallowedValues))
		}
	}
	// TODO(lschumacher): verify https://github.com/terraform-providers/terraform-provider-google/issues/2648
	// is available.
	if lp.InheritFromParent {
		result += "inherit_from_parent = true"
	}
	result += " }"

	return result, nil
}

// tfFormatBooleanPolicy returns the Terraform string for a Boolean policy.
func tfFormatBooleanPolicy(bp OrganizationPolicyBooleanPolicy) string {
	return fmt.Sprintf(`boolean_policy {
		enforced = %t
		}`, bp.Enforced)
}

// tfFormatConstraintPolicy returns the terraform string for the constraint based on whether
// its a list or a boolean policy.  Since we can't use pointers, we don't
// really have a good way to ensure that only one of these is set.
func tfFormatConstraintPolicy(constraint OrganizationPolicyConstraint) (string, error) {
	lp := constraint.ListPolicy
	if len(lp.AllowedValues) > 0 || len(lp.DisallowedValues) > 0 || lp.AllValues != "" {
		return tfFormatListPolicyResource(lp)
	}
	return tfFormatBooleanPolicy(constraint.BooleanPolicy), nil
}

// tfResourceID returns the resource ID in the correct TF plugin format.  The gcp plugin for terraform uses an inconsistent format
// for these ids (ie, only using kind for the folders), hence the need for this code.
func tfResourceID(ctx context.Context, client Client, ps OrganizationPolicySpec) (string, error) {
	kind := ps.ResourceRef.Kind
	res, err := ResourceID(ctx, client, kind, ps.ResourceRef.Name)
	if err != nil {
		return "", err
	}
	var result string
	switch kind {
	case OrganizationKind, ProjectKind:
		result = res
	case FolderKind:
		// The Terraform GCP adapter is maddingly inconsistent here.
		result = fmt.Sprintf("folders/%s", res)
	}
	return result, nil
}

// TFResourceConfig converts the Project's Spec struct into Terraform config string.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (op *OrganizationPolicy) TFResourceConfig(ctx context.Context, c Client) (string, error) {
	ps := op.Spec
	kind := ps.ResourceRef.Kind

	// Make sure we like the kind value.
	switch kind {
	case OrganizationKind, FolderKind, ProjectKind:
		// ok.
	default:
		return "", errors.Errorf("invalid kind: %s", kind)
	}

	resourceID, err := tfResourceID(ctx, c, ps)
	if err != nil {
		return "", err
	}

	var tfs strings.Builder
	for i, constraint := range ps.Constraints {
		constraintPolicyName := fmt.Sprintf("bespin-%s-policy-%d", kind, i)
		var constraintPolicy string
		constraintPolicy, err = tfFormatConstraintPolicy(constraint)
		if err != nil {
			return "", errors.Wrapf(err, "bad constraint: %s", constraintPolicyName)
		}
		_, err = fmt.Fprintf(&tfs, `
	resource "%s" "%s" {
		%s = "%s"
		constraint = "%s"
		%s
	}
	`,
			tfPolicyResourceName(kind),
			constraintPolicyName,
			tfPolicyResourceKeyName(kind),
			resourceID,
			constraint.Constraint,
			constraintPolicy)
		if err != nil {
			return "", err
		}
	}
	formatString, err := printer.Format([]byte(tfs.String()))
	if err != nil {
		glog.Errorf("Unexpected error formatting Terraform string: %v", err)
		// not sure this is the right thing to do?
		return tfs.String(), nil
	}

	return string(formatString), nil

}

// TFImportConfig returns an empty Terraform OrganizationPolicy resource block used for Terraform import.
// It implements terraform.Resource interface.
func (op *OrganizationPolicy) TFImportConfig() string {
	switch op.Kind {
	case ProjectKind:
		return `resource "google_project_organization_policy" "project_policy" {}`
	case OrganizationKind:
		return `resource "google_organization_policy" "org_policy" {}`
	case FolderKind:
		return `resource "google_folder_organization_policy" "folder_policy" {}`
	}
	return "" // panic?
}

// TFResourceAddr returns the address of this OrganizationPolicy resource in terraform config.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (op *OrganizationPolicy) TFResourceAddr() string {
	switch op.Kind {
	case ProjectKind:
		return "google_project_organization_policy.project_policy"
	case OrganizationKind:
		return "google_organization_policy.org_policy"
	case FolderKind:
		return "google_folder_organization_policy.folder_policy"
	}
	return "" // panic?
}

// ID returns the OrganizationPolicy ID from GCP.  OrganizationPolicy doesn't
// really have a separate ID, its just the resource Id of the attachment point.
// It implements the github.com/google/nomos/pkg/bespin-controllers/terraform.Resource interface.
func (op *OrganizationPolicy) ID() string {
	return op.Spec.ResourceRef.Name
}

// OrganizationPolicyList contains a list of OrganizationPolicy
type OrganizationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OrganizationPolicy{}, &OrganizationPolicyList{})
}
