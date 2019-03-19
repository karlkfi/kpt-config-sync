package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
type UnsupportedRepoSpecVersion struct {
	id.Resource
	Version string
}

// UnsupportedRepoSpecVersionCode is the error code for UnsupportedRepoSpecVersion
const UnsupportedRepoSpecVersionCode = "1027"

func init() {
	status.Register(UnsupportedRepoSpecVersionCode, UnsupportedRepoSpecVersion{})
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
