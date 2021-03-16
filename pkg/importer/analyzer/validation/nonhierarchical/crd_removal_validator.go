package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedCRDRemovalErrorCode is the error code for UnsupportedCRDRemovalError
const UnsupportedCRDRemovalErrorCode = "1047"

var unsupportedCRDRemovalError = status.NewErrorBuilder(UnsupportedCRDRemovalErrorCode)

// UnsupportedCRDRemovalError reports than a CRD was removed, but its corresponding CRs weren't.
func UnsupportedCRDRemovalError(resource id.Resource) status.Error {
	return unsupportedCRDRemovalError.
		Sprintf("Custom Resources MUST be removed in the same commit as their corresponding " +
			"CustomResourceDefinition. To fix, remove this Custom Resource or re-add the CRD.").
		BuildWithResources(resource)
}
