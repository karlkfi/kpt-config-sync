package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScoped validates the given FileObject as a cluster-scoped resource to
// ensure it does not have a namespace or a namespace selector.
func ClusterScoped(obj ast.FileObject) status.Error {
	if obj.GetNamespace() != "" {
		return nonhierarchical.IllegalNamespaceOnClusterScopedResourceError(&obj)
	}
	if hasNamespaceSelector(obj) {
		return nonhierarchical.IllegalNamespaceSelectorAnnotationError(&obj)
	}
	return nil
}

// ClusterScopedForNamespaceReconciler immediately throws an error for any given
// cluster-scoped resource because a namespace reconciler should only manage
// namespace-scoped resources.
func ClusterScopedForNamespaceReconciler(obj ast.FileObject) status.Error {
	return shouldBeInRootErr(&obj)
}

// NamespaceScoped validates the given FileObject as a namespace-scoped resource
// to ensure it does not have both namespace and namespace selector.
func NamespaceScoped(obj ast.FileObject) status.Error {
	if obj.GetNamespace() != "" && hasNamespaceSelector(obj) {
		return nonhierarchical.NamespaceAndSelectorResourceError(obj)
	}
	return nil
}

func hasNamespaceSelector(obj ast.FileObject) bool {
	_, ok := obj.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
	return ok
}

func shouldBeInRootErr(resource client.Object) status.ResourceError {
	return nonhierarchical.BadScopeErrBuilder.
		Sprintf("Resources in namespace Repos must be Namespace-scoped type, but objects of type %v are Cluster-scoped. Move %s to the Root repo.",
			resource.GetObjectKind().GroupVersionKind(), resource.GetName()).
		BuildWithResources(resource)
}
