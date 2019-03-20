package vet

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
	utilrepo "github.com/google/nomos/pkg/util/repo"
)

// IllegalSystemResourcePlacementErrorCode is the error code for IllegalSystemResourcePlacementError
const IllegalSystemResourcePlacementErrorCode = "1033"

func init() {
	r := ast.NewFileObject(utilrepo.Default(), cmpath.FromSlash("namespaces/foo/repo.yaml"))
	status.Register(IllegalSystemResourcePlacementErrorCode, IllegalSystemResourcePlacementError{
		Resource: &r,
	})
}

// IllegalSystemResourcePlacementError reports that a configmanagement.gke.io object has been defined outside of system/
type IllegalSystemResourcePlacementError struct {
	id.Resource
}

var _ id.ResourceError = &IllegalSystemResourcePlacementError{}

// Error implements error
func (e IllegalSystemResourcePlacementError) Error() string {
	return status.Format(e,
		"Resources of the below kind MUST NOT be declared outside %[1]s/:\n"+
			"%[2]s",
		repo.SystemDir, id.PrintResource(e))
}

// Code implements Error
func (e IllegalSystemResourcePlacementError) Code() string {
	return IllegalSystemResourcePlacementErrorCode
}

// Resources implements ResourceError
func (e IllegalSystemResourcePlacementError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
