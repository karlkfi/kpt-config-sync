package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedCRDRemovalErrorCode is the error code for UnsupportedCRDRemovalError
const UnsupportedCRDRemovalErrorCode = "1047"

func init() {
	status.AddExamples(UnsupportedCRDRemovalErrorCode, UnsupportedCRDRemovalError(customResourceDefinition()))
}

var unsupportedCRDRemovalError = status.NewErrorBuilder(UnsupportedCRDRemovalErrorCode)

// UnsupportedCRDRemovalError reports than a CRD was removed, but its corresponding CRs weren't.
func UnsupportedCRDRemovalError(resource id.Resource) status.Error {
	return unsupportedCRDRemovalError.WithResources(resource).Errorf(
		"Removing a CRD and leaving the corresponding Custom Resources in the repo is disallowed. To fix, " +
			"remove the CRD along with the Custom Resources.")
}
