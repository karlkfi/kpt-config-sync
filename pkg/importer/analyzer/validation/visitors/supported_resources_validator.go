package visitors

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewSupportedClusterResourcesValidator initializes a ValidatorVisitor that
// ensures all ClusterObjects are supported resources.
func NewSupportedClusterResourcesValidator() *visitor.ValidatorVisitor {
	ensureSupported := func(o *ast.ClusterObject) status.MultiError {
		if !hierarchyconfig.AllowedInHierarchyConfigs(o.GroupVersionKind().GroupKind()) {
			return syntax.UnsupportedObjectError(o)
		}
		return nil
	}

	return visitor.NewClusterObjectValidator(ensureSupported)
}
