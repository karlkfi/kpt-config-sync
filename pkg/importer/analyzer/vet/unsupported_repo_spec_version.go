package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/repo"
)

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
type UnsupportedRepoSpecVersion struct {
	id.Resource
	Version string
}

// UnsupportedRepoSpecVersionCode is the error code for UnsupportedRepoSpecVersion
const UnsupportedRepoSpecVersionCode = "1027"

func init() {
	o := ast.NewFileObject(repo.Default(), cmpath.FromSlash("system/repo/yaml"))
	status.Register(UnsupportedRepoSpecVersionCode, UnsupportedRepoSpecVersion{
		Resource: &o,
		Version:  "0.0.0",
	})
}

var _ status.ResourceError = UnsupportedRepoSpecVersion{}

// Error implements error
func (e UnsupportedRepoSpecVersion) Error() string {
	return status.Format(e,
		"Unsupported Repo spec.version: %q. Must use version %q",
		e.Version, repo.CurrentVersion)
}

// Code implements Error
func (e UnsupportedRepoSpecVersion) Code() string { return UnsupportedRepoSpecVersionCode }

// Resources implements ResourceError
func (e UnsupportedRepoSpecVersion) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e UnsupportedRepoSpecVersion) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
