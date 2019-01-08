package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// SystemKindValidator ensures only the allowed set of Kinds appear in system/
var SystemKindValidator = &FileObjectValidator{
	ValidateFn: func(object ast.FileObject) error {
		switch object.Object.(type) {
		case *v1alpha1.Repo:
		case *v1alpha1.Sync:
		default:
			return veterrors.IllegalKindInSystemError{ResourceID: &object}
		}
		return nil
	},
}
