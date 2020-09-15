package nonhierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IllegalNamespaceOnClusterScopedResourceErrorCode represents a cluster-scoped resource illegally
// declaring metadata.namespace.
const IllegalNamespaceOnClusterScopedResourceErrorCode = "1052"

var illegalNamespaceOnClusterScopedResourceErrorBuilder = status.NewErrorBuilder(IllegalNamespaceOnClusterScopedResourceErrorCode)

// IllegalNamespaceOnClusterScopedResourceError reports that a cluster-scoped resource MUST NOT declare metadata.namespace.
func IllegalNamespaceOnClusterScopedResourceError(resource id.Resource) status.Error {
	return illegalNamespaceOnClusterScopedResourceErrorBuilder.
		Sprint("cluster-scoped resources MUST NOT declare metadata.namespace").
		BuildWithResources(resource)
}

// MissingNamespaceOnNamespacedResourceErrorCode represents a namespace-scoped resource NOT declaring
// metadata.namespace.
const MissingNamespaceOnNamespacedResourceErrorCode = "1053"

var missingNamespaceOnNamespacedResourceErrorBuilder = status.NewErrorBuilder(MissingNamespaceOnNamespacedResourceErrorCode)

// NamespaceAndSelectorResourceError reports that a namespace-scoped resource illegally declares both metadata.namespace
// and has the namespace-selector annotation.
func NamespaceAndSelectorResourceError(resource id.Resource) status.Error {
	return missingNamespaceOnNamespacedResourceErrorBuilder.
		Sprintf("namespace-scoped resources MUST NOT declare both metadata.namespace and "+
			"metadata.annotations.%s", v1.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}

// MissingNamespaceOnNamespacedResourceError reports a namespace-scoped resource MUST declare metadata.namespace.
// when parsing in non-hierarchical mode.
func MissingNamespaceOnNamespacedResourceError(resource id.Resource) status.Error {
	return missingNamespaceOnNamespacedResourceErrorBuilder.
		Sprintf("namespace-scoped resources MUST either declare either metadata.namespace or "+
			"metadata.annotations.%s", v1.NamespaceSelectorAnnotationKey).
		BuildWithResources(resource)
}

// ScopeValidator returns errors for resources with illegal metadata.namespace
// declarations.
//
// If the object is namespace-scoped and does not declare a NamespaceSelector,
// it is automatically assigned to the "default" Namespace.
func ScopeValidator(scoper discovery.Scoper) Validator {
	return PerObjectValidator(func(o ast.FileObject) status.Error {
		// Skip the validation when it is a Kptfile.
		// Kptfile is only for client side. A ResourceGroup CR will be generated from it
		// in subsequent program in pkg/parse.
		if o.GroupVersionKind().GroupKind() == kinds.KptFile().GroupKind() {
			return nil
		}
		isNamespaced, err := scoper.GetObjectScope(o)
		if err != nil {
			return err
		}

		if isNamespaced {
			// namespace-scoped resources must declare either metadata.namespace or the
			// NamespaceSelector annotation when in nonhierarchical mode.
			hasNamespace := o.GetNamespace() != ""
			_, hasNamespaceSelector := o.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]

			if hasNamespace && hasNamespaceSelector {
				return NamespaceAndSelectorResourceError(o)
			}
			if !hasNamespace && !hasNamespaceSelector {
				o.SetNamespace(metav1.NamespaceDefault)
			}
		} else if o.GetNamespace() != "" {
			return IllegalNamespaceOnClusterScopedResourceError(&o)
		}

		return nil
	})
}
