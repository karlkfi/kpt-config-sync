package vet

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
	"github.com/google/nomos/pkg/policyimporter/id"
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

var _ id.ResourceError = &UnsupportedRepoSpecVersion{}

// Error implements error
func (e UnsupportedRepoSpecVersion) Error() string {
	return status.Format(e,
		"Unsupported Repo spec.version: %[2]q. Must use version \"0.1.0\"\n\n"+
			"%[1]s",
		id.PrintResource(e), e.Version)
}

// Code implements Error
func (e UnsupportedRepoSpecVersion) Code() string { return UnsupportedRepoSpecVersionCode }

// Resources implements ResourceError
func (e UnsupportedRepoSpecVersion) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
