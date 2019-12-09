package ast

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPartialCopy(t *testing.T) {
	n := TreeNode{
		Labels: map[string]string{"foo": "bar"},
	}
	nCopy := n.PartialCopy()
	// Original TreeNode should not change if labels or annotations are modified
	// on the copy.
	nCopy.Labels["foo"] = "baz"
	if diff := cmp.Diff(n.Labels, map[string]string{"foo": "bar"}); diff != "" {
		t.Errorf("Actual and expected annotations didn't match: %v", diff)
	}
}
