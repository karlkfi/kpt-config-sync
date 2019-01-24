package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

var (
	resourceQuotaModes = map[v1alpha1.HierarchyModeType]bool{
		v1alpha1.HierarchyModeDefault:           true,
		v1alpha1.HierarchyModeHierarchicalQuota: true,
		v1alpha1.HierarchyModeInherit:           true,
		v1alpha1.HierarchyModeNone:              true,
	}
	otherTypesModes = map[v1alpha1.HierarchyModeType]bool{
		v1alpha1.HierarchyModeDefault: true,
		v1alpha1.HierarchyModeInherit: true,
		v1alpha1.HierarchyModeNone:    true,
	}
)

// NewInheritanceValidatorFactory creates a validator factory that validates the inheritance setting
// of all Kinds defined in Syncs.
func NewInheritanceValidatorFactory() ValidatorFactory {
	return ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
		if sync.GroupVersionKind() == kinds.ResourceQuota() {
			return errIfNotAllowed(sync, resourceQuotaModes)
		}
		return errIfNotAllowed(sync, otherTypesModes)
	}}
}

// errIfNotAllowed returns an error if the kindSync has an inheritance mode which is not allowed for that Kind.
func errIfNotAllowed(sync FileGroupVersionKindHierarchySync, allowed map[v1alpha1.HierarchyModeType]bool) error {
	if allowed[sync.HierarchyMode] {
		return nil
	}
	return vet.IllegalHierarchyModeError{
		Sync:          sync,
		HierarchyMode: sync.HierarchyMode,
		Allowed:       allowed,
	}
}
