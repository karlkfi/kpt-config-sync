package validation

import (
	"testing"

	"github.com/google/nomos/pkg/bespin/kinds"
	nomoskinds "github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
)

func TestUniqueIAMValidatorVisitTreeNode(t *testing.T) {
	var testCases = []struct {
		name       string
		objects    []ast.FileObject
		shouldFail bool
	}{
		{
			name: "empty",
		},
		{
			name: "one IAMPolicy",
			objects: []ast.FileObject{
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
			},
		},
		{
			name: "one IAMPolicy and one Role",
			objects: []ast.FileObject{
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
				asttesting.NewFakeFileObject(nomoskinds.Role(), ""),
			},
		},
		{
			name: "two IAMPolicies same version",
			objects: []ast.FileObject{
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v1"), ""),
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v1"), ""),
			},
			shouldFail: true,
		},
		{
			name: "two IAMPolicies different version",
			objects: []ast.FileObject{
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v1"), ""),
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion("v2"), ""),
			},
			shouldFail: true,
		},
		{
			name: "three IAMPolicies",
			objects: []ast.FileObject{
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
				asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), ""),
			},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objects := make([]*ast.NamespaceObject, len(tc.objects))
			for i, object := range tc.objects {
				objects[i] = &ast.NamespaceObject{FileObject: object}
			}

			node := &ast.TreeNode{
				Objects: objects,
			}

			v := NewUniqueIAMValidator()

			v.VisitTreeNode(node)

			if tc.shouldFail {
				vettesting.ExpectErrors([]string{vet.UndocumentedErrorCode}, v.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, v.Error(), t)
			}
		})
	}
}
