// nolint
package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
)

const (
	// nolint
	ReadyConditionType = "Ready"
)

// nolint
type Condition struct {
	// Last time the condition transitioned from one status to another.
	// nolint
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`

	// Human-readable message indicating details about last transition.
	// nolint
	Message string `json:"message,omitempty"`

	// Unique, one-word, CamelCase reason for the condition's last
	// transition.
	// nolint
	Reason string `json:"reason,omitempty"`

	// Status is the status of the condition. Can be True, False, Unknown.
	// nolint
	Status v1.ConditionStatus `json:"status,omitempty"`

	// Type is the type of the condition.
	// nolint
	Type string `json:"type,omitempty"`
}
