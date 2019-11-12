package syntax

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// NewDisallowedCRDValidator validates that Nomos CRDs are not in the repo.
func NewDisallowedCRDValidator() *visitor.ValidatorVisitor {
	return visitor.NewClusterObjectValidator(func(o *ast.ClusterObject) status.MultiError {
		return IllegalCRD(o.FileObject)
	})
}

// IllegalCRD returns an error if o is a CRD of a Nomos type.
func IllegalCRD(o ast.FileObject) status.Error {
	if o.GroupVersionKind() != kinds.CustomResourceDefinition() {
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
	return unsupportedObjectError.WithResources(resource).Errorf(
		"%s cannot configure this resource. To fix, remove this resource from the repo.",
		configmanagement.ProductName)
}
