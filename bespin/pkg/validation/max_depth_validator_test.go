package validation

import (
	"testing"

	"github.com/google/nomos/bespin/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

func organization(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Organization().WithVersion(""), path)
}

func folder(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Folder().WithVersion(""), path)
}

func project(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Project().WithVersion(""), path)
}

func TestMaxDepthValidatorVisitTreeNode(t *testing.T) {
	test := vt.ObjectsValidatorTest{
		Validator: NewMaxFolderDepthValidator,
		ErrorCode: vet.UndocumentedErrorCode,
		TestCases: []vt.ObjectsValidatorTestCase{
			{
				Name:    "depth 0",
			},
			{
				Name:    "depth 1",
				Objects: []ast.FileObject{
					folder("hierarchy/1"),
				},
			},
			{
				Name:    "depth 4",
				Objects: []ast.FileObject{
					folder("hierarchy/1"),
					folder("hierarchy/1/2"),
					folder("hierarchy/1/2/3"),
					folder("hierarchy/1/2/3/4"),
				},
			},
			{
				Name:       "depth 5",
				Objects: []ast.FileObject{
					folder("hierarchy/1"),
					folder("hierarchy/1/2"),
					folder("hierarchy/1/2/3"),
					folder("hierarchy/1/2/3/4"),
					folder("hierarchy/1/2/3/4/5"),
				},
				ShouldFail: true,
			},
			{
				Name:    "depth 4 project",
				Objects: []ast.FileObject{
					folder("hierarchy/1"),
					folder("hierarchy/1/2"),
					folder("hierarchy/1/2/3"),
					folder("hierarchy/1/2/3/4"),
					project("hierarchy/1/2/3/4/project"),
				},
			},
			{
				Name:       "depth 5 project",
				Objects: []ast.FileObject{
					folder("hierarchy/1"),
					folder("hierarchy/1/2"),
					folder("hierarchy/1/2/3"),
					folder("hierarchy/1/2/3/4"),
					folder("hierarchy/1/2/3/4/5"),
					project("hierarchy/1/2/3/4/5/project"),
				},
				ShouldFail: true,
			},
			{
				Name:    "org depth 0",
				Objects: []ast.FileObject{
					organization("hierarchy/org"),
				},
			},
			{
				Name:    "org depth 1",
				Objects: []ast.FileObject{
					organization("hierarchy/org"),
					folder("hierarchy/org/1"),
				},
			},
			{
				Name:    "org depth 4",
				Objects: []ast.FileObject{
					organization("hierarchy/org"),
					folder("hierarchy/org/1"),
					folder("hierarchy/org/1/2"),
					folder("hierarchy/org/1/2/3"),
					folder("hierarchy/org/1/2/3/4"),
				},
			},
			{
				Name:       "org depth 5",
				Objects: []ast.FileObject{
					organization("hierarchy/org"),
					folder("hierarchy/org/1"),
					folder("hierarchy/org/1/2"),
					folder("hierarchy/org/1/2/3"),
					folder("hierarchy/org/1/2/3/4"),
					folder("hierarchy/org/1/2/3/4/5"),
				},
				ShouldFail: true,
			},
			{
				Name:    "org depth 4 project",
				Objects: []ast.FileObject{
					organization("hierarchy/org"),
					folder("hierarchy/org/1"),
					folder("hierarchy/org/1/2"),
					folder("hierarchy/org/1/2/3"),
					folder("hierarchy/org/1/2/3/4"),
					project("hierarchy/org/1/2/3/4/project"),
				},
			},
			{
				Name:       "org depth 5 project",
				Objects: []ast.FileObject{
					organization("hierarchy/org"),
					folder("hierarchy/org/1"),
					folder("hierarchy/org/1/2"),
					folder("hierarchy/org/1/2/3"),
					folder("hierarchy/org/1/2/3/4"),
					folder("hierarchy/org/1/2/3/4/5"),
					project("hierarchy/org/1/2/3/4/5/project"),
				},
				ShouldFail: true,
			},
		},
	}

	test.Run(t)
}
