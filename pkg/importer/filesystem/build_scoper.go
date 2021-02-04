package filesystem

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/vet"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// BuildScoper constructs a Scoper to use to get the scope of any type declared
// in the repository.
//
// BuildScoper only contacts the API Server if both:
// 1. useAPIServer is true, and
// 2. there are declared types we can't establish the scope of.
// TODO(b/172446570): Refactor to make this accept a list of AddAPIResourcesFn
//   so it has fewer arguments and the logic is more separable.
func BuildScoper(
	dc utildiscovery.ServerResourcer,
	// if false, don't contact the API Server.
	useAPIServer bool,
	// The cached API Resources.
	addCachedAPIResources vet.AddCachedAPIResourcesFn,
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

	// It is possible that the cached API Resources conflicts with declared CRDs.
	// For this edge case, the declared CRD takes precedence as, once synced,
	// the new api-resources.txt will eventually be updated to reflect this change.
	err := addCachedAPIResources(&scoper)
	if err != nil {
		return scoper, nil, err
	}
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
	// Build a new Scoper from the cluster's API resource lists and any previously
	// synced CRDs.
	newScoper := utildiscovery.Scoper{}

	// List the APIResources from the API Server and add them.
	lists, discoveryErr := utildiscovery.GetResources(dc)
	if discoveryErr != nil {
		return scoper, nil, discoveryErr
	}

	if addListsErr := newScoper.AddAPIResourceLists(lists); addListsErr != nil {
		return scoper, nil, addListsErr
	}

	// Add previously declared CRDs second, as they may overwrite ones on the API Server.
	syncedCRDs, err := getSyncedCRDs()
	if err != nil {
		return scoper, nil, err
	}
	newScoper.AddCustomResources(syncedCRDs)

	// Finally add the other scoper on top which includes core resources, cached
	// API resources, and currently declared CRDs.
	newScoper.AddScoper(scoper)
	return newScoper, syncedCRDs, nil
}
