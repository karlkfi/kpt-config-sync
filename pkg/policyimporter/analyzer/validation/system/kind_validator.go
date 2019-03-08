package system

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewKindValidator returns a validator that ensures only allowed resource kinds are defined in
// system/.
func NewKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) *status.MultiError {
		switch o.Object.(type) {
		case *v1.Repo:
		case *v1.HierarchyConfig:
		default:
			return status.From(vet.IllegalKindInSystemError{Resource: o})
		}
		return nil
	})
}
