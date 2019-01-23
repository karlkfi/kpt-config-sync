package vet

import "github.com/google/nomos/pkg/policyimporter/id"

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
type UnsupportedRepoSpecVersion struct {
	id.Resource
	Version string
}

// UnsupportedRepoSpecVersionCode is the error code for UnsupportedRepoSpecVersion
const UnsupportedRepoSpecVersionCode = "1027"

func init() {
	register(UnsupportedRepoSpecVersionCode, nil, "")
}

// Error implements error
func (e UnsupportedRepoSpecVersion) Error() string {
	return format(e,
		"Unsupported Repo spec.version: %[2]q. Must use version \"0.1.0\"\n\n"+
			"%[1]s",
		id.PrintResource(e), e.Version)
}

// Code implements Error
func (e UnsupportedRepoSpecVersion) Code() string { return UnsupportedRepoSpecVersionCode }
