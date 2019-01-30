package visitors

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
)

func fakeRole(dir string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Role(), dir+"/role.yaml")
}

func TestUniqueDirectoryValidator(t *testing.T) {
	testCases := []struct {
		name       string
		root       *ast.Root
		shouldFail bool
	}{
		{
			name: "empty",
		},
		{
			name: "just namespaces/",
			root: treetesting.BuildTree(fakeRole("namespaces")),
		},
		{
			name: "one dir",
			root: treetesting.BuildTree(fakeRole("namespaces/foo")),
		},
		{
			name:       "subdirectory of self",
			root:       treetesting.BuildTree(fakeRole("namespaces/foo/foo")),
			shouldFail: true,
		},
		{
			name:       "deep subdirectory of self",
			root:       treetesting.BuildTree(fakeRole("namespaces/foo/bar/foo")),
			shouldFail: true,
		},
		{
			name:       "child of different directories",
			root:       treetesting.BuildTree(fakeRole("namespaces/bar/foo"), fakeRole("namespaces/qux/foo")),
			shouldFail: true,
		},
		{
			name: "directory with two children",
			root: treetesting.BuildTree(fakeRole("namespaces/bar/foo"), fakeRole("namespaces/bar/qux")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			var v ast.Visitor = NewUniqueDirectoryValidator()
			tc.root.Accept(v)

			if tc.shouldFail {
				vettesting.ExpectErrors([]string{vet.DuplicateDirectoryNameErrorCode}, v.Error(), t)
			} else {
				vettesting.ExpectErrors([]string{}, v.Error(), t)
			}
		})
	}
}
