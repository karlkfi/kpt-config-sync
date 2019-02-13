package visitors

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewSupportedClusterResourcesValidator initializes a ValidatorVisitor that ensures all ClusterObjects are supported
// resources.
func NewSupportedClusterResourcesValidator() *visitor.ValidatorVisitor {
	ensureSupported := func(o *ast.ClusterObject) error {
		if !hierarchyconfig.AllowedInHierarchyConfigs(o.GroupVersionKind().GroupKind()) {
			return vet.UnsupportedObjectError{Resource: o}
		}
		return nil
	}

	return visitor.NewClusterObjectValidator(ensureSupported)
}
