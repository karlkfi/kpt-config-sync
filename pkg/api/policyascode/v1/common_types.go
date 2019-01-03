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
	corev1 "k8s.io/api/core/v1"
)

// Finalizer defines the k8s finalizer for bespin CRDs.
const Finalizer string = "finalizer.bespin.dev"

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

// IAMPolicyStatus defines the observed state of IAMPolicy
type IAMPolicyStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
}

// OrganizationPolicySpec defines the desired state of OrganizationPolicy
type OrganizationPolicySpec struct {
	ResourceRef corev1.ObjectReference         `json:"resourceRef"`
	Constraints []OrganizationPolicyConstraint `json:"constraints"`
}

// OrganizationPolicyStatus defines the observed state of OrganizationPolicy
type OrganizationPolicyStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
}

// OrganizationPolicyConstraint is the Schema for Constraints in OrganizationPolicySpec
// Note that ListPolicy and BooleanPolicy are mutually exclusive.
// TODO(b/121393215): add validation on creation/import.
type OrganizationPolicyConstraint struct {
	Constraint    string                          `json:"constraint"`
	ListPolicy    OrganizationPolicyListPolicy    `json:"listPolicy,omitempty"`
	BooleanPolicy OrganizationPolicyBooleanPolicy `json:"booleanPolicy,omitempty"`
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
