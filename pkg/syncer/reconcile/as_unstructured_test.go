package reconcile

import (
	"encoding/json"
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAsUnstructured_AddsStatus(t *testing.T) {
	testCases := []struct {
		name string
		obj  client.Object
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
		obj  client.Object
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

func TestAsUnstructuredSanitized_DeepCopy(t *testing.T) {
	wantName := "foo"

	testCases := []struct {
		name string
		obj  client.Object
	}{
		{
			name: "Namespace as object",
			obj:  &corev1.Namespace{TypeMeta: fake.ToTypeMeta(kinds.Namespace())},
		},
		{
			name: "Namespace as unstructured",
			obj:  newUnstructured(kinds.Namespace()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.obj.SetName(wantName)

			u, err := AsUnstructuredSanitized(tc.obj)
			if err != nil {
				t.Error(err)
				t.Fatalf("unable to convert %T to Unstructured", tc.obj)
			}
			// Verify the name was unchanged by conversion to unstructured.
			if u.GetName() != wantName {
				t.Errorf("got name %q, want name %q", u.GetName(), wantName)
			}

			// Modify the original name and verify the unstructured name is still unchanged.
			tc.obj.SetName("bar")
			if u.GetName() != wantName {
				t.Errorf("got name %q, want name %q", u.GetName(), wantName)
			}
		})
	}
}

func newUnstructured(gvk schema.GroupVersionKind) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	return u
}
