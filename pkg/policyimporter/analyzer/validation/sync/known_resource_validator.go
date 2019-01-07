package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/meta"
)

// KnownResourceValidatorFactory adds errors for unknown resources which are not explicitly unsupported.
func KnownResourceValidatorFactory(apiInfo *meta.APIInfo) ValidatorFactory {
	return ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
		gvk := sync.GroupVersionKind()
		if !apiInfo.Exists(gvk) {
			versions := apiInfo.AllowedVersions(gvk.GroupKind())
			if versions == nil {
				return veterrors.UnknownResourceInSyncError{SyncID: sync}
			}
			return veterrors.UnknownResourceVersionInSyncError{SyncID: sync, Allowed: versions}
		}
		return nil
	}}
}
