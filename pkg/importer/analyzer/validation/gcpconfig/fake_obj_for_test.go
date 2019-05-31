package gcpconfig

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/kinds"
)

func orgFileObj(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Organization().WithVersion(""), path)
}

func folderFileObj(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Folder().WithVersion(""), path)
}

func projectFileObj(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Project().WithVersion(""), path)
}

func iamPolicyFileObj(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.IAMPolicy().WithVersion(""), path)
}

func orgPolicyFileObj(path string) ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.OrganizationPolicy().WithVersion(""), path)
}
