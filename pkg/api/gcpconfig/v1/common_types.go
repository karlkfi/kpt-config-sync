package v1

import (
	corev1 "k8s.io/api/core/v1"
)

// Finalizer defines the k8s finalizer for bespin CRDs.
const Finalizer string = "finalizer.bespin.dev"

// ResourceState contains the resource state. An example of a gcp project state
// may look like the below:
// id              = folders/986145150667
// create_time     = 2019-01-15T01:47:19.518Z
// display_name    = zlu-org-folder1
// lifecycle_state = ACTIVE
// name            = folders/986145150667
// parent          = organizations/975672035171
// Please refer to each resource's Terraform (Google provider) doc for all the fields
// that may appear in the state. For example, GCP project:
// https://www.terraform.io/docs/providers/google/r/google_project.html
type ResourceState map[string]string

// Condition defines a set of common observed fields for GCP resources. This
// is a copy-n-paste from CNRM code repo:
// https://cnrm.git.corp.google.com/cnrm/+/master/pkg/apis/k8s/v1alpha1/condition_types.go
type Condition struct {
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// Human-readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Status is the status of the condition. Can be True, False, Unknown.
	Status corev1.ConditionStatus `json:"status,omitempty"`
	// Type is the type of the condition.
	Type string `json:"type,omitempty"`
}

// Status defines the observed state of resources
type Status struct {
	Conditions []Condition `json:"conditions,omitempty"`
}

// IAMPolicyBinding is the Schema for Bindings in IAMPolicySpec
type IAMPolicyBinding struct {
	// The validation pattern is based on https://cloud.google.com/iam/reference/rest/v1/Binding
	// +kubebuilder:validation:Pattern=^((user|serviceAccount|group|domain):.+)|allAuthenticatedUsers|allUsers$
	Members []string `json:"members"`
	// The validation pattern is based on https://cloud.google.com/iam/reference/rest/v1/Binding
	// Usually role looks like e.g. "roles/viewer", "roles/editor", or "roles/owner" etc.
	// For custom role however, it supports project and organization level roles, see
	// https://cloud.google.com/iam/docs/creating-custom-roles,
	// e.g. "projects/project_id/roles/viewer" and "organizations/organization_id/roles/editor".
	// +kubebuilder:validation:Pattern=(^roles|^(projects|organizations)/.+/roles)/[\w\.]+$
	Role string `json:"role"`
}

// IAMPolicySpec defines the desired state of IAMPolicy
type IAMPolicySpec struct {
	ResourceRef corev1.ObjectReference `json:"resourceRef"`
	Bindings    []IAMPolicyBinding     `json:"bindings"`
}

// OrganizationPolicySpec defines the desired state of OrganizationPolicy
type OrganizationPolicySpec struct {
	ResourceRef corev1.ObjectReference         `json:"resourceRef"`
	Constraints []OrganizationPolicyConstraint `json:"constraints"`
}

// OrganizationPolicyConstraint is the Schema for Constraints in OrganizationPolicySpec
// Note that ListPolicy and BooleanPolicy are mutually exclusive.
// The current list of constraints: https://cloud.google.com/resource-manager/docs/organization-policy/org-policy-constraints
// TODO(b/121393215): add validation on creation/import.
type OrganizationPolicyConstraint struct {
	Constraint string                       `json:"constraint"`
	ListPolicy OrganizationPolicyListPolicy `json:"listPolicy,omitempty"`
	// +optional
	BooleanPolicy *OrganizationPolicyBooleanPolicy `json:"booleanPolicy,optional"`
	// +optional
	RestoreDefaultPolicy bool `json:"restoreDefaultPolicy,omitempty"`
}

// OrganizationPolicyListPolicy is the Schema for ListPolicy in OrganizationPolicyConstraint
type OrganizationPolicyListPolicy struct {
	// +kubebuilder:validation:Pattern=^((is|under):)?(organizations|folders|projects)/
	AllowedValues []string `json:"allowedValues,omitempty"`
	// +kubebuilder:validation:Pattern=^((is|under):)?(organizations|folders|projects)/
	DisallowedValues []string `json:"disallowedValues,omitempty"`
	// +kubebuilder:validation:Pattern=^(ALLOW|DENY)$
	AllValues         string `json:"allValues,omitempty"`
	InheritFromParent bool   `json:"inheritFromParent,omitempty"`
}

// OrganizationPolicyBooleanPolicy is the Schema for BooleanPolicy in OrganizationPolicyConstraint
type OrganizationPolicyBooleanPolicy struct {
	Enforced bool `json:"enforced"`
}
