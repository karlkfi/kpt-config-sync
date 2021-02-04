package validation

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var topLevelDirectoryOverrides = map[schema.GroupVersionKind]string{
	kinds.Repo():            repo.SystemDir,
	kinds.HierarchyConfig(): repo.SystemDir,

	kinds.Cluster():         repo.ClusterRegistryDir,
	kinds.ClusterSelector(): repo.ClusterRegistryDir,

	kinds.Namespace():         repo.NamespacesDir,
	kinds.NamespaceSelector(): repo.NamespacesDir,
}

const unknownDir = ""

// getExpectedTopLevelDir returns the top-level directory we expect this object to be in,
// or an error if we were unable to determine in which one it belongs.
func getExpectedTopLevelDir(scoper discovery.Scoper, o id.Resource) (string, status.Error) {
	gvk := o.GroupVersionKind()
	if override, hasOverride := topLevelDirectoryOverrides[gvk]; hasOverride {
		return override, nil
	}

	scope, err := scoper.GetObjectScope(o)
	if err != nil {
		return unknownDir, err
	}
	switch scope {
	case discovery.NamespaceScope:
		return repo.NamespacesDir, nil
	case discovery.ClusterScope:
		return repo.ClusterDir, nil
	default:
		return unknownDir, nil
	}
}

// NewTopLevelDirectoryValidator ensures Namespaces and namespace-scoped objects are in namespaces/,
// and cluster-scoped objects are in cluster/.
//
// Returns an UnknownObjectKindError if unable to determine which top-level directory
// the resource should live. This happens when the resource is neither present
// on the APIServer nor has a CRD defined.
func NewTopLevelDirectoryValidator(scoper discovery.Scoper, errOnUnknown bool) nonhierarchical.Validator {
	return nonhierarchical.PerObjectValidator(func(o ast.FileObject) status.Error {
		return validateTopLevelDirectory(scoper, o, errOnUnknown)
	})
}

func validateTopLevelDirectory(scoper discovery.Scoper, o ast.FileObject, errOnUnknown bool) status.Error {
	expectedTopLevelDir, err := getExpectedTopLevelDir(scoper, o)
	if err != nil {
		if errOnUnknown {
			return err
		}
		glog.V(6).Infof("ignored error due to --no-api-server-check: %s", err)
	}
	if expectedTopLevelDir == unknownDir {
		// We don't know for sure which directory this should be in, and we can't
		// check a cluster.
		return nil
	}

	sourcePath := o.Relative.OSPath()
	if cmpath.RelativeSlash(sourcePath).Split()[0] == expectedTopLevelDir {
		return nil
	}

	switch expectedTopLevelDir {
	case repo.SystemDir:
		return ShouldBeInSystemError(o)
	case repo.ClusterRegistryDir:
		return ShouldBeInClusterRegistryError(o)
	case repo.ClusterDir:
		return ShouldBeInClusterError(o)
	case repo.NamespacesDir:
		return ShouldBeInNamespacesError(o)
	default:
		return status.InternalErrorf("unhandled top level directory: %q", expectedTopLevelDir)
	}
}

// IncorrectTopLevelDirectoryErrorCode is the error code for IllegalKindInClusterError
const IncorrectTopLevelDirectoryErrorCode = "1039"

var incorrectTopLevelDirectoryErrorBuilder = status.NewErrorBuilder(IncorrectTopLevelDirectoryErrorCode)

// ShouldBeInSystemError reports that an object belongs in system/.
func ShouldBeInSystemError(resource id.Resource) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Repo and HierarchyConfig configs MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.SystemDir, resource.GroupVersionKind().Kind, repo.SystemDir).
		BuildWithResources(resource)
}

// ShouldBeInClusterRegistryError reports that an object belongs in clusterregistry/.
func ShouldBeInClusterRegistryError(resource id.Resource) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Cluster and ClusterSelector configs MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.ClusterRegistryDir, resource.GroupVersionKind().Kind, repo.ClusterRegistryDir).
		BuildWithResources(resource)
}

// ShouldBeInClusterError reports that an object belongs in cluster/.
func ShouldBeInClusterError(resource id.Resource) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Cluster-scoped configs except Namespaces MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.ClusterDir, resource.GroupVersionKind().Kind, repo.ClusterDir).
		BuildWithResources(resource)
}

// ShouldBeInNamespacesError reports that an object belongs in namespaces/.
func ShouldBeInNamespacesError(resource id.Resource) status.Error {
	return incorrectTopLevelDirectoryErrorBuilder.
		Sprintf("Namespace-scoped and Namespace configs MUST be declared in `%s/`. "+
			"To fix, move the %s to `%s/`.", repo.NamespacesDir, resource.GroupVersionKind().Kind, repo.NamespacesDir).
		BuildWithResources(resource)
}
