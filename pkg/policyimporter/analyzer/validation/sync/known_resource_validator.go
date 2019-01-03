package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/meta"
)

// KnownResourceValidatorFactory adds errors for unknown resources which are not explicitly unsupported.
func KnownResourceValidatorFactory(apiInfo *meta.APIInfo) ValidatorFactory {
	return ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
		gvk := sync.GroupVersionKind()
		if !apiInfo.Exists(gvk) {
			versions := apiInfo.AllowedVersions(gvk.GroupKind())
			if versions == nil {
				return vet.UnknownResourceInSyncError{SyncID: sync}
			}
			return vet.UnknownResourceVersionInSyncError{SyncID: sync, Allowed: versions}
		}
		return nil
	}}
}
