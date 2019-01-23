package visitors

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
)

func buildTree(dirs []string, t *testing.T) *ast.TreeNode {
	tree := filesystem.NewDirectoryTree()
	for _, dir := range dirs {
		tree.AddDir(nomospath.NewFakeRelative(dir), node.AbstractNamespace)
	}
	eb := multierror.Builder{}
	result := tree.Build(&eb)
	if eb.HasErrors() {
		t.Fatal(eb.Build())
	}
	return result
}

func TestUniqueDirectoryValidator(t *testing.T) {
	testCases := []struct {
		name       string
		dirs       []string
		shouldFail bool
	}{
		{
			name: "empty",
		},
		{
			name: "just namespaces/",
			dirs: []string{"namespaces"},
		},
		{
			name: "one dir",
			dirs: []string{"namespaces", "namespaces/foo"},
		},
		{
			name:       "subdirectory of self",
			dirs:       []string{"namespaces", "namespaces/foo", "namespaces/foo/foo"},
			shouldFail: true,
		},
		{
			name:       "deep subdirectory of self",
			dirs:       []string{"namespaces", "namespaces/foo", "namespaces/foo/bar", "namespaces/foo/bar/foo"},
			shouldFail: true,
		},
		{
			name:       "child of different directories",
			dirs:       []string{"namespaces", "namespaces/foo", "namespaces/foo/bar", "namespaces/qux", "namespaces/qux/bar"},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := ast.Root{Tree: buildTree(tc.dirs, t)}

			var v ast.Visitor = NewUniqueDirectoryValidator()
			root.Accept(v)

			if tc.shouldFail {
				vettesting.ExpectErrors([]string{vet.DuplicateDirectoryNameErrorCode}, v.Error(), t)
			} else {
				vettesting.ExpectErrors([]string{}, v.Error(), t)
			}
		})
	}
}
