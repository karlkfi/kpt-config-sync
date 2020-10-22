package filesystem

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// BuildScoper constructs a Scoper to use to get the scope of any type declared
// in the repository.
//
// BuildScoper only contacts the API Server if both:
// 1. useAPIServer is true, and
// 2. there are declared types we can't establish the scope of.
func BuildScoper(
	dc utildiscovery.ServerResourcer,
	// if false, don't contact the API Server.
	useAPIServer bool,
	// The list of declared FileObjects. Used to determine if API Server calls are necessary.
	fileObjects []ast.FileObject,
	// The list of CRDs declared in the repository.
	declaredCRDs []*v1beta1.CustomResourceDefinition,
	// The callback to get the currently-synced CRDs. Only called if necessary.
	getSyncedCRDs GetSyncedCRDs,
) (utildiscovery.Scoper, []*v1beta1.CustomResourceDefinition, status.MultiError) {
	// Initialize the scoper with the default set of Kubernetes resources and the
	// declared CRDs.
	scoper := utildiscovery.CoreScoper()
	scoper.AddCustomResources(declaredCRDs)

	// If we don't need to check the API Server because we have all the required
	// type information, or the user has passed --no-api-server-check, don't
	// call the API Server.
	if useAPIServer && !scoper.HasScopesFor(fileObjects) {
		// We're allowed to talk to the API Server, and we don't have the scopes
		// for some types.
		return addSyncedCRDs(scoper, dc, getSyncedCRDs)
	}

	// Note that we return declaredCRDs as syncedCRDs in this case. We've
	// determined that either the scope for all types is present in the
	// repository or the caller has explicitly asked to not check the API Server.
	// This ensures later checks that depend on the synced CRDs are effectively
	// no-ops.
	return scoper, declaredCRDs, nil
}

func addSyncedCRDs(scoper utildiscovery.Scoper, dc utildiscovery.ServerResourcer, getSyncedCRDs GetSyncedCRDs) (utildiscovery.Scoper, []*v1beta1.CustomResourceDefinition, status.MultiError) {
	// Add CRDs from the ClusterConfig first since those may overwrites ones on
	// the API Server in the future.
	syncedCRDs, err := getSyncedCRDs()
	if err != nil {
		return nil, nil, err
	}
	scoper.AddCustomResources(syncedCRDs)

	// List the APIResources from the API Server.
	lists, discoveryErr := utildiscovery.GetResources(dc)
	if discoveryErr != nil {
		return nil, nil, discoveryErr
	}

	// Add resources from the API Server last.
	if addListsErr := scoper.AddAPIResourceLists(lists); addListsErr != nil {
		return nil, nil, addListsErr
	}

	return scoper, syncedCRDs, nil
}
