package vet

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
)

// UnsupportedRepoSpecVersionCode is the error code for UnsupportedRepoSpecVersion
const UnsupportedRepoSpecVersionCode = "1027"

func init() {
	o := ast.NewFileObject(repo.Default(), cmpath.FromSlash("system/repo/yaml"))
	status.AddExamples(UnsupportedRepoSpecVersionCode, UnsupportedRepoSpecVersion(
		&o,
		"0.0.0",
	))
}

var unsupportedRepoSpecVersion = status.NewErrorBuilder(UnsupportedRepoSpecVersionCode)

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
func UnsupportedRepoSpecVersion(resource id.Resource, version string) status.Error {
	return unsupportedRepoSpecVersion.WithResources(resource).Errorf(
		"Unsupported Repo spec.version: %q. Must use version %q",
		version, repo.CurrentVersion)
}
