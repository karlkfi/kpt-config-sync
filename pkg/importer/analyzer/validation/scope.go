package validation

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewTopLevelDirectoryValidator ensures Namespaces and namespace-scoped objects are in namespaces/,
// and cluster-scoped objects are in cluster/.
//
// Returns an UnknownObjectError if unable to determine which top-level directory
// the resource should live. This happens when the resource is neither present
// on the APIServer nor has a CRD defined.
func NewTopLevelDirectoryValidator(scoper discovery.Scoper) nonhierarchical.Validator {
	return nonhierarchical.PerObjectValidator(func(o ast.FileObject) status.Error {
		return validateTopLevelDirectory(scoper, o)
	})
}

func validateTopLevelDirectory(scoper discovery.Scoper, o ast.FileObject) status.Error {
	gvk := o.GroupVersionKind()
	scope := scoper.GetScope(gvk.GroupKind())
	topLevelDir := o.Path.Split()[0]

	if isIgnored(gvk) {
		return nil
	}

	if isClusterScopedAllowedInNamespaces(gvk) || scope == discovery.NamespaceScope {
		// Only Namespace-scoped object and Namespaces are in the namespaces/ directory.
		if topLevelDir != repo.NamespacesDir {
			return ShouldBeInNamespacesError(topLevelDir, o)
		}
		return nil
	}

	if scope == discovery.ClusterScope {
		if topLevelDir != repo.ClusterDir {
			return ShouldBeInClusterError(topLevelDir, o)
		}
		return nil
	}

	return UnknownObjectError(o)
}

func isClusterScopedAllowedInNamespaces(gvk schema.GroupVersionKind) bool {
	return gvk == kinds.Namespace() ||
		gvk == kinds.NamespaceSelector()
}

func isIgnored(gvk schema.GroupVersionKind) bool {
	return gvk == kinds.HierarchicalQuota()
}

// UnknownObjectErrorCode is the error code for UnknownObjectError
const UnknownObjectErrorCode = "1021" // Impossible to create consistent example.

var unknownObjectError = status.NewErrorBuilder(UnknownObjectErrorCode)

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
func UnknownObjectError(resource id.Resource) status.Error {
	return unknownObjectError.
		Sprint("No CustomResourceDefinition is defined for the resource in the cluster. " +
			"\nResource types that are not native Kubernetes objects must have a CustomResourceDefinition.").
		BuildWithResources(resource)
}

// IncorrectTopLevelDirectoryErrorCode is the error code for IllegalKindInClusterError
const IncorrectTopLevelDirectoryErrorCode = "1039"

var incorrectTopLevelDirectoryErrorBuilder = status.NewErrorBuilder(IncorrectTopLevelDirectoryErrorCode)

// ShouldBeInNamespacesError reports that an object belongs in namespaces/.
func ShouldBeInNamespacesError(dir string, resource id.Resource) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Namespace-scoped and Namespace configs MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", dir, resource.GroupVersionKind().Kind, repo.NamespacesDir).
		BuildWithResources(resource)
}

// ShouldBeInClusterError reports that an object belongs in cluster/.
func ShouldBeInClusterError(dir string, resource id.Resource) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Cluster-scoped configs except Namespaces MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", dir, resource.GroupVersionKind().Kind, repo.ClusterDir).
		BuildWithResources(resource)
}
