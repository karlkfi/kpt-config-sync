package validation

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IncorrectTopLevelDirectoryErrorCode is the error code for IllegalKindInClusterError
const IncorrectTopLevelDirectoryErrorCode = "1039"

var incorrectTopLevelDirectoryErrorBuilder = status.NewErrorBuilder(IncorrectTopLevelDirectoryErrorCode)

// ShouldBeInSystemError reports that an object belongs in system/.
func ShouldBeInSystemError(resource client.Object) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Repo and HierarchyConfig configs MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.SystemDir, resource.GetObjectKind().GroupVersionKind().Kind, repo.SystemDir).
		BuildWithResources(resource)
}

// ShouldBeInClusterRegistryError reports that an object belongs in clusterregistry/.
func ShouldBeInClusterRegistryError(resource client.Object) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Cluster and ClusterSelector configs MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.ClusterRegistryDir, resource.GetObjectKind().GroupVersionKind().Kind, repo.ClusterRegistryDir).
		BuildWithResources(resource)
}

// ShouldBeInClusterError reports that an object belongs in cluster/.
func ShouldBeInClusterError(resource client.Object) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Cluster-scoped configs except Namespaces MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.ClusterDir, resource.GetObjectKind().GroupVersionKind().Kind, repo.ClusterDir).
		BuildWithResources(resource)
}

// ShouldBeInNamespacesError reports that an object belongs in namespaces/.
func ShouldBeInNamespacesError(resource client.Object) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Namespace-scoped and Namespace configs MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.NamespacesDir, resource.GetObjectKind().GroupVersionKind().Kind, repo.NamespacesDir).
		BuildWithResources(resource)
}
