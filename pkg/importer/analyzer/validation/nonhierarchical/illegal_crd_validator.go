package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UnsupportedObjectErrorCode is the error code for UnsupportedObjectError
const UnsupportedObjectErrorCode = "1043"

var unsupportedObjectError = status.NewErrorBuilder(UnsupportedObjectErrorCode)

// UnsupportedObjectError reports than an unsupported object is in the namespaces/ sub-directories or clusters/ directory.
func UnsupportedObjectError(resource client.Object) status.Error {
	return unsupportedObjectError.
		Sprintf("%s does not allow configuring CRDs in the `%s` APIGroup. To fix, please use a different APIGroup.",
			configmanagement.ProductName, v1.SchemeGroupVersion.Group).
		BuildWithResources(resource)
}
