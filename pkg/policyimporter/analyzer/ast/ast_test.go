package ast

import (
	"testing"

	"github.com/go-test/deep"
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
	if diff := deep.Equal(orig.Labels, map[string]string{"foo": "bar"}); diff != nil {
		t.Errorf("Actual and expected annotations didn't match: %v", diff)
	}

	copy.Annotations["foo"] = "hux"
	if diff := deep.Equal(orig.Annotations, map[string]string{"foo": "bar"}); diff != nil {
		t.Errorf("Actual and expected annotations didn't match: %v", diff)
	}
}
