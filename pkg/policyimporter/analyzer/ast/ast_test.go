package ast

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPartialCopy(t *testing.T) {
	orig := TreeNode{
		Labels:      map[string]string{"foo": "bar"},
		Annotations: map[string]string{"foo": "bar"},
	}
	copy := orig.PartialCopy()
	// Original TreeNode should not change if labels or annotations are modified
	// on the copy.
	copy.Labels["foo"] = "baz"
	if diff := cmp.Diff(orig.Labels, map[string]string{"foo": "bar"}); diff != "" {
		t.Errorf("Actual and expected annotations didn't match: %v", diff)
	}

	copy.Annotations["foo"] = "hux"
	if diff := cmp.Diff(orig.Annotations, map[string]string{"foo": "bar"}); diff != "" {
		t.Errorf("Actual and expected annotations didn't match: %v", diff)
	}
}
