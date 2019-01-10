package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

var (
	resourceQuotaIfInheritanceEnabled = map[v1alpha1.HierarchyModeType]bool{
		v1alpha1.HierarchyModeDefault:           true,
		v1alpha1.HierarchyModeHierarchicalQuota: true,
		v1alpha1.HierarchyModeInherit:           true,
		v1alpha1.HierarchyModeNone:              true,
	}
	othersIfInheritanceEnabled = map[v1alpha1.HierarchyModeType]bool{
		v1alpha1.HierarchyModeDefault: true,
		v1alpha1.HierarchyModeInherit: true,
		v1alpha1.HierarchyModeNone:    true,
	}
	inheritanceDisabled = map[v1alpha1.HierarchyModeType]bool{
		v1alpha1.HierarchyModeDefault: true,
	}
)

// NewInheritanceValidatorFactory creates a validator factory for the passed Repo that validates the
// inheritance setting of all Kinds defined in Syncs. If the passed repo is Nil, returns the
// disabled validator.
func NewInheritanceValidatorFactory(repo *v1alpha1.Repo) ValidatorFactory {
	if repo == nil {
		return nilValidatorFactory
	}
	if repo.Spec.ExperimentalInheritance {
		return newInheritanceEnabledValidator()
	}
	return newInheritanceDisabledValidator()
}

// newInheritanceDisabledValidator implements validation for when inheritance is disabled.
func newInheritanceDisabledValidator() ValidatorFactory {
	return ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
		return errIfNotAllowed(sync, inheritanceDisabled)
	}}
}

// newInheritanceEnabledValidator implements validation for when inheritance is enabled.
func newInheritanceEnabledValidator() ValidatorFactory {
	return ValidatorFactory{fn: func(sync FileGroupVersionKindHierarchySync) error {
		if sync.GroupVersionKind() == kinds.ResourceQuota() {
			return errIfNotAllowed(sync, resourceQuotaIfInheritanceEnabled)
		}
		return errIfNotAllowed(sync, othersIfInheritanceEnabled)
	}}
}

// errIfNotAllowed returns an error if the kindSync has an inheritance mode which is not allowed for that Kind.
func errIfNotAllowed(sync FileGroupVersionKindHierarchySync, allowed map[v1alpha1.HierarchyModeType]bool) error {
	if allowed[sync.HierarchyMode] {
		return nil
	}
	return veterrors.IllegalHierarchyModeError{
		Sync:          sync,
		HierarchyMode: sync.HierarchyMode,
		Allowed:       allowed,
	}
}
