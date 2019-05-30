package system

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// NewKindValidator returns a validator that ensures only allowed resource kinds are defined in
// system/.
func NewKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) status.MultiError {
		switch o.Object.(type) {
		case *v1.Repo:
		case *v1.HierarchyConfig:
		default:
			return status.From(vet.IllegalKindInSystemError(o))
		}
		return nil
	})
}
