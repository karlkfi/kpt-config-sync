package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// IllegalCRDValidator forbids CRDs declaring Nomos types.
var IllegalCRDValidator = PerObjectValidator(illegalCRD)

// IllegalCRD returns an error if o is a CRD of a Nomos type.
func illegalCRD(o ast.FileObject) status.Error {
	if o.GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
		return nil
	}

	crd, err := clusterconfig.AsCRD(o.Object)
	if err != nil {
		return err
	}
	if crd.Spec.Group == v1.SchemeGroupVersion.Group {
		return UnsupportedObjectError(&o)
	}
	return nil
}

// UnsupportedObjectErrorCode is the error code for UnsupportedObjectError
const UnsupportedObjectErrorCode = "1043"

var unsupportedObjectError = status.NewErrorBuilder(UnsupportedObjectErrorCode)

// UnsupportedObjectError reports than an unsupported object is in the namespaces/ sub-directories or clusters/ directory.
func UnsupportedObjectError(resource id.Resource) status.Error {
	return unsupportedObjectError.
		Sprintf("%s does not allow configuring CRDs in the `%s` APIGroup. To fix, please use a different APIGroup.",
			configmanagement.ProductName, v1.SchemeGroupVersion.Group).
		BuildWithResources(resource)
}
