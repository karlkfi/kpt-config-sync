package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SystemKindValidator ensures only the allowed set of Kinds appear in system/
var SystemKindValidator = &ObjectValidator{
	validate: func(source string, object runtime.Object) error {
		switch o := object.(type) {
		case *v1alpha1.Repo:
		case *corev1.ConfigMap:
		case *v1alpha1.Sync:
		default:
			return vet.IllegalKindInSystemError{Source: source, GroupVersionKind: o.GetObjectKind().GroupVersionKind()}
		}
		return nil
	},
}
