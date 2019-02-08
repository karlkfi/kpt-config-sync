package system

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// NewKindValidator returns a validator that ensures only allowed resource kinds are defined in
// system/.
func NewKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) error {
		switch o.Object.(type) {
		case *v1alpha1.Repo:
		case *v1alpha1.Sync:
		case *v1alpha1.HierarchyConfig:
		default:
			return vet.IllegalKindInSystemError{Resource: o}
		}
		return nil
	})
}
