package queue

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestWasDeleted(t *testing.T) {
	testCases := []struct {
		name string
		obj  core.Object
	}{
		{
			"object with no annotations",
			fake.ConfigMapObject(),
		},
		{
			"object with an annotation",
			fake.ConfigMapObject(core.Annotation("hello", "world")),
		},
		{
			"object with explicitly empty annotations",
			fake.ConfigMapObject(core.Annotations(map[string]string{})),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First verify that the object is not detected as deleted.
			if WasDeleted(tc.obj) {
				t.Errorf("object was incorrectly detected as deleted: %v", tc.obj)
			}
			// Next mark the object as deleted and verify that it is now detected.
			deletedObj := MarkDeleted(tc.obj)
			if !WasDeleted(deletedObj) {
				t.Errorf("deleted object was not detected: %v", tc.obj)
			}
		})
	}
}
