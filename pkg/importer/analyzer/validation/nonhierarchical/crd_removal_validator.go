package nonhierarchical

import (
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UnsupportedCRDRemovalErrorCode is the error code for UnsupportedCRDRemovalError
const UnsupportedCRDRemovalErrorCode = "1047"

var unsupportedCRDRemovalError = status.NewErrorBuilder(UnsupportedCRDRemovalErrorCode)

// UnsupportedCRDRemovalError reports than a CRD was removed, but its corresponding CRs weren't.
func UnsupportedCRDRemovalError(resource client.Object) status.Error {
	return unsupportedCRDRemovalError.
		Sprintf("Custom Resources MUST be removed in the same commit as their corresponding " +
			"CustomResourceDefinition. To fix, remove this Custom Resource or re-add the CRD.").
		BuildWithResources(resource)
}
