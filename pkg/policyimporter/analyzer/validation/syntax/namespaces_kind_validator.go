package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// NamespacesKindValidator ensures only the allowed set of Kinds appear in namespaces/
var NamespacesKindValidator = &FileObjectValidator{
	ValidateFn: func(object ast.FileObject) error {
		switch object.Object.(type) {
		case *v1alpha1.NamespaceSelector:
			return veterrors.IllegalKindInNamespacesError{Resource: &object}
		default:
		}
		return nil
	},
}
