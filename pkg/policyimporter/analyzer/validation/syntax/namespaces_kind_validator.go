package syntax

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/runtime"
)

// NamespacesKindValidator ensures only the allowed set of Kinds appear in namespaces/
var NamespacesKindValidator = &ObjectValidator{
	validate: func(source string, object runtime.Object) error {
		switch o := object.(type) {
		case *v1alpha1.NamespaceSelector:
			return vet.IllegalKindInNamespacesError{Source: source, GroupVersionKind: o.GetObjectKind().GroupVersionKind()}
		default:
		}
		return nil
	},
}
