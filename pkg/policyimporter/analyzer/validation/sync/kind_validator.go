package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// KindValidatorFactory ensures that only supported Resource Kinds are declared in Syncs.
var KindValidatorFactory = ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
	if allowedInSyncs(sync.GroupVersionKind()) {
		return nil
	}
	return veterrors.UnsupportedResourceInSyncError{
		Sync: sync,
	}
}}

// allowedInSyncs returns true if the passed GVK is allowed to be declared in Syncs.
func allowedInSyncs(gvk schema.GroupVersionKind) bool {
	return !unsupportedSyncResources()[gvk] && (gvk.Group != policyhierarchy.GroupName)
}

// unsupportedSyncResources returns a map of each type where syncing is explicitly not supported.
func unsupportedSyncResources() map[schema.GroupVersionKind]bool {
	return map[schema.GroupVersionKind]bool{
		kinds.CustomResourceDefinition(): true,
		kinds.Namespace():                true,
	}
}
