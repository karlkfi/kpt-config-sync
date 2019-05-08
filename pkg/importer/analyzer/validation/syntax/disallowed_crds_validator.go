package syntax

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// NewDisallowedCRDsValidator validates that Nomos CRDs are not in the repo.
func NewDisallowedCRDsValidator() *visitor.ValidatorVisitor {
	return visitor.NewClusterObjectValidator(func(o *ast.ClusterObject) status.MultiError {
		if o.GroupVersionKind() != kinds.CustomResourceDefinition() {
			return nil
		}

		crd, err := clusterconfig.AsCRD(o.Object)
		if err != nil {
			return status.From(status.ResourceWrap(err, "could not deserialize CRD", o))
		}
		if crd.Spec.Group == v1.SchemeGroupVersion.Group {
			return status.From(vet.UnsupportedObjectError{Resource: o})
		}
		return nil
	})
}
