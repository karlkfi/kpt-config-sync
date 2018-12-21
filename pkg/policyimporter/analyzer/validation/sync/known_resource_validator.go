package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/meta"
)

// KnownResourceValidatorFactory adds errors for unknown resources which are not explicitly unsupported.
func KnownResourceValidatorFactory(apiInfo *meta.APIInfo) ValidatorFactory {
	return ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
		gvk := sync.GroupVersionKind
		if !apiInfo.Exists(gvk) {
			return vet.UnknownResourceInSyncError{Source: sync.Source, GVK: gvk}
		}
		return nil
	}}
}
