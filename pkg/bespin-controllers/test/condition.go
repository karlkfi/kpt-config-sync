package test

import (
	"testing"

	condition "github.com/google/nomos/pkg/api/policyascode/v1"
)

// AssertReadyCondition checks that the given conditions slice contains a Ready condition.
func AssertReadyCondition(t *testing.T, conditions []condition.Condition) {
	t.Helper()
	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, instead have %v", len(conditions))
	}
	readyCondition := conditions[0]
	if readyCondition.Type != "Ready" {
		t.Errorf("readyCondition type mismatch: got '%v', want '%v'", readyCondition.Type, "Ready")
	}
	if readyCondition.LastTransitionTime == "" {
		t.Errorf("readyCondition last transition time is empty string, expected value")
	}
	if readyCondition.Status != "True" {
		t.Errorf("status value mismatch: got '%v', want '%v'", readyCondition.Status, "True")
	}
}
