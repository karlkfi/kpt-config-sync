package validation

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
)

func TestMaxDepthValidatorVisitTreeNode(t *testing.T) {
	var testCases = []struct {
		name       string
		path       string
		shouldFail bool
	}{
		{
			name: "depth 0",
			path: "hierarchy/",
		},
		{
			name: "depth 1",
			path: "hierarchy/1",
		},
		{
			name: "depth 4",
			path: "hierarchy/1/2/3/4",
		},
		{
			name:       "depth 5",
			path:       "hierarchy/1/2/3/4/5",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.TreeNode{
				Relative: nomospath.NewFakeRelative(tc.path),
			}

			v := NewMaxDepthValidator()

			v.VisitTreeNode(node)

			if tc.shouldFail {
				veterrorstest.ExpectErrors([]string{veterrors.UndocumentedErrorCode}, v.Error(), t)
			} else {
				veterrorstest.ExpectErrors(nil, v.Error(), t)
			}
		})
	}
}
