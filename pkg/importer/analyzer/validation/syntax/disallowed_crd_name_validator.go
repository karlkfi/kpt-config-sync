package syntax

import (
	"fmt"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// NewDisallowedCRDNameValidator returns a validator that validates CRDs have valid names.
func NewDisallowedCRDNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewClusterObjectValidator(func(o *ast.ClusterObject) status.MultiError {
		if o.GroupVersionKind() != kinds.CustomResourceDefinition() {
			return nil
		}

		crd, err := clusterconfig.AsCRD(o.Object)
		if err != nil {
			return status.From(status.ResourceWrap(err, "could not deserialize CRD", o))
		}

		expectedName := fmt.Sprintf("%s.%s", crd.Spec.Names.Plural, crd.Spec.Group)
		if crd.Name != expectedName {
			return status.From(vet.InvalidCRDNameError(o))
		}

		return nil
	})
}
