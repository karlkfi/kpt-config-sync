package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	corev1 "k8s.io/api/core/v1"
)

// SystemKindValidator ensures only the allowed set of Kinds appear in system/
var SystemKindValidator = &FileObjectValidator{
	ValidateFn: func(object ast.FileObject) error {
		switch object.Object.(type) {
		case *v1alpha1.Repo:
		case *corev1.ConfigMap:
		case *v1alpha1.Sync:
		default:
			return vet.IllegalKindInSystemError{Source: object.Source, GroupVersionKind: object.GroupVersionKind()}
		}
		return nil
	},
}
