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
var IllegalCRDValidator = PerObjectValidator(IllegalCRD)

// IllegalCRD returns an error if o is a CRD of a Nomos type.
func IllegalCRD(o ast.FileObject) status.Error {
	if o.GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
		return nil
	}

	crd, err := clusterconfig.AsCRD(o.Object)
	if err != nil {
		return status.ResourceWrap(err, "could not deserialize CRD", &o)
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
		Sprintf("%s cannot configure CRDs in the %q APIGroup",
			configmanagement.ProductName, v1.SchemeGroupVersion.Group).
		BuildWithResources(resource)
}
