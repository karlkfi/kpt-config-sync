package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		if isResourceQuota(sync.GroupVersionKind) {
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
	return vet.IllegalHierarchyModeError{
		Source:           sync.Source,
		GroupVersionKind: sync.GroupVersionKind,
		HierarchyMode:    sync.HierarchyMode,
		Allowed:          allowed,
	}
}

func isResourceQuota(gvk schema.GroupVersionKind) bool {
	return gvk.Kind == "ResourceQuota" && gvk.Group == corev1.GroupName
}
