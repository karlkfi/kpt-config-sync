package system

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
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
			return IllegalKindInSystemError(o)
		}
		return nil
	})
}

// IllegalKindInSystemErrorCode is the error code for IllegalKindInSystemError
const IllegalKindInSystemErrorCode = "1024"

var illegalKindInSystemError = status.NewErrorBuilder(IllegalKindInSystemErrorCode)

// IllegalKindInSystemError reports that an object has been illegally defined in system/
func IllegalKindInSystemError(resource id.Resource) status.Error {
	return illegalKindInSystemError.
		Sprintf("Configs of this Kind may not be declared in the `%s/` directory of the repo:", repo.SystemDir).
		BuildWithResources(resource)
}
