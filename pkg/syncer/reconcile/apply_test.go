package reconcile

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestEqual(t *testing.T) {
	testcases := []struct {
		name          string
		dryrunStatus  *unstructured.Unstructured
		currentStatus *unstructured.Unstructured
		equal         bool
	}{
		{
			name:          "exactly the same object",
			dryrunStatus:  fake.UnstructuredObject(kinds.Namespace(), core.Name("test")),
			currentStatus: fake.UnstructuredObject(kinds.Namespace(), core.Name("test")),
			equal:         true,
		},
		{
			name:          "same object with different generations, different timestamp",
			dryrunStatus:  fake.UnstructuredObject(kinds.Namespace(), core.Name("test"), core.Generation(1), core.CreationTimeStamp(metav1.Time{})),
			currentStatus: fake.UnstructuredObject(kinds.Namespace(), core.Name("test"), core.Generation(2), core.CreationTimeStamp(metav1.Now())),
			equal:         true,
		},
		{
			name:         "same object with status",
			dryrunStatus: fake.UnstructuredObject(kinds.Namespace(), core.Name("test")),
			currentStatus: fake.UnstructuredObject(kinds.Namespace(), core.Name("test"),
				func(o client.Object) {
					u := o.(*unstructured.Unstructured)
					err := unstructured.SetNestedField(u.Object, "Active", "status", "phase")
					if err != nil {
						t.Fatal("failed to set the status field")
					}
				}),
			equal: true,
		},
		{
			name:          "same object with different labels",
			dryrunStatus:  fake.UnstructuredObject(kinds.Namespace(), core.Name("test"), core.Label("key", "val1")),
			currentStatus: fake.UnstructuredObject(kinds.Namespace(), core.Name("test"), core.Label("key", "val2")),
			equal:         false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := equal(tc.dryrunStatus, tc.currentStatus)
			if actual != tc.equal {
				t.Errorf("equal should be %v, but got %v", tc.equal, actual)
			}
		})
	}
}
