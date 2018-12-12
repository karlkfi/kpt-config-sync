package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
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

// InheritanceValidator ensures only valid inheritance modes are used in the repository.
type InheritanceValidator struct {
	Repo *v1alpha1.Repo
}

// Validate adds errors for each Kind defined in a Sync with an illegal inheritance mode.
func (v InheritanceValidator) Validate(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	v.inheritanceValidator().Validate(objects, errorBuilder)
}

func (v InheritanceValidator) inheritanceValidator() *validator {
	return &validator{
		validate: func(sync kindSync) error {
			allowed := v.allowedModes(sync)
			if !allowed[sync.hierarchy] {
				return vet.IllegalHierarchyModeError{Object: sync.sync, GVK: sync.gvk, Mode: sync.hierarchy, Allowed: allowed}
			}
			return nil
		},
	}
}

func (v InheritanceValidator) allowedModes(sync kindSync) map[v1alpha1.HierarchyModeType]bool {
	if v.Repo.Spec.ExperimentalInheritance {
		if sync.gvk.Kind == "ResourceQuota" && sync.gvk.Group == corev1.GroupName {
			return resourceQuotaIfInheritanceEnabled
		}
		return othersIfInheritanceEnabled
	}
	return inheritanceDisabled
}
