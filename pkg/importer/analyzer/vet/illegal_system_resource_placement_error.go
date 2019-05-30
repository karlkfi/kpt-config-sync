package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	utilrepo "github.com/google/nomos/pkg/util/repo"
)

// IllegalSystemResourcePlacementErrorCode is the error code for IllegalSystemResourcePlacementError
const IllegalSystemResourcePlacementErrorCode = "1033"

func init() {
	r := ast.NewFileObject(utilrepo.Default(), cmpath.FromSlash("namespaces/foo/repo.yaml"))
	status.AddExamples(IllegalSystemResourcePlacementErrorCode, IllegalSystemResourcePlacementError(&r))
}

var illegalSystemResourcePlacementError = status.NewErrorBuilder(IllegalSystemResourcePlacementErrorCode)

// IllegalSystemResourcePlacementError reports that a configmanagement.gke.io object has been defined outside of system/
func IllegalSystemResourcePlacementError(resource id.Resource) status.Error {
	return illegalSystemResourcePlacementError.WithResources(resource).Errorf(
		"A config of the below kind MUST NOT be declared outside %[1]s/:",
		repo.SystemDir)
}
