package reconcile

import (
	"encoding/json"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestAsUnstructured_AddsStatus(t *testing.T) {
	testCases := []struct {
		name string
		obj  core.Object
	}{
		{
			name: "Namespace",
			obj:  &corev1.Namespace{TypeMeta: fake.ToTypeMeta(kinds.Namespace())},
		},
		{
			name: "Service",
			obj:  &corev1.Service{TypeMeta: fake.ToTypeMeta(kinds.Service())},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := AsUnstructured(tc.obj)
			if err != nil {
				t.Error(err)
				t.Fatalf("unable to convert %T to Unstructured", tc.obj)
			}

			// Yes, we don't like this behavior.
			// These checks validate the bug.
			if _, hasStatus := u.Object["status"]; !hasStatus {
				jsn, _ := json.MarshalIndent(u, "", "  ")
				t.Log(string(jsn))
				t.Error("got .status undefined, want defined")
			}

			metadata := u.Object["metadata"].(map[string]interface{})
			if _, hasCreationTimestamp := metadata["creationTimestamp"]; !hasCreationTimestamp {
				jsn, _ := json.MarshalIndent(u, "", "  ")
				t.Log(string(jsn))
				t.Error("got .metadata.creationTimestamp undefined, want defined")
			}
		})
	}
}

func TestAsUnstructuredSanitized_DoesNotAddStatus(t *testing.T) {
	testCases := []struct {
		name string
		obj  core.Object
	}{
		{
			name: "Namespace",
			obj:  &corev1.Namespace{TypeMeta: fake.ToTypeMeta(kinds.Namespace())},
		},
		{
			name: "Service",
			obj:  &corev1.Service{TypeMeta: fake.ToTypeMeta(kinds.Service())},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := AsUnstructuredSanitized(tc.obj)
			if err != nil {
				t.Error(err)
				t.Fatalf("unable to convert %T to Unstructured", tc.obj)
			}

			if _, hasStatus := u.Object["status"]; hasStatus {
				jsn, _ := json.MarshalIndent(u, "", "  ")
				t.Log(string(jsn))
				t.Error("got .status defined, want undefined")
			}

			metadata := u.Object["metadata"].(map[string]interface{})
			if _, hasCreationTimestamp := metadata["creationTimestamp"]; hasCreationTimestamp {
				jsn, _ := json.MarshalIndent(u, "", "  ")
				t.Log(string(jsn))
				t.Error("got .status defined, want undefined")
			}
		})
	}
}
