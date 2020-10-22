package parse

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

func noClusterScopeValidator(scoper discovery.Scoper) nonhierarchical.Validator {
	return nonhierarchical.PerObjectValidator(func(o ast.FileObject) status.Error {
		scope, err := scoper.GetObjectScope(o)
		if err != nil {
			// We don't know the scope of this object since:
			// 1) It isn't a default type.
			// 2) There is no CRD for the type on the cluster.
			//
			// As long as the Root repo is configured properly (has all required CRDs)
			// this will self-resolve in seconds. As this is a transient expected
			// error, we can expect customers to notice.
			return err
		}
		if scope != discovery.NamespaceScope {
			// This can only happen if there is actually a problem - i.e. a type we
			// know is cluster-scoped is in a Namespace repo.
			return shouldBeInRootErr(o)
		}
		return nil
	})
}

func shouldBeInRootErr(resource id.Resource) status.ResourceError {
	return badScopeErrBuilder.
		Sprintf("Resources in namespace Repos must be Namespace-scoped type, but objects of type %v are Cluster-scoped. Move %s to the Root repo.",
			resource.GroupVersionKind(), resource.GetName()).
		BuildWithResources(resource)
}
