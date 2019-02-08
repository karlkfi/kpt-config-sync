package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
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

// NewInheritanceValidator returns a visitor that validates the inheritance setting
// of all GroupKinds defined across HierarchyConfigs.
func NewInheritanceValidator() *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) error {
		switch h := o.Object.(type) {
		case *v1alpha1.HierarchyConfig:
			for _, gkc := range NewFileHierarchyConfig(h, o.Relative).flatten() {
				if err := ValidateInheritance(gkc); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ValidateInheritance returns an error if the HierarchyModeType is invalid for the GroupKind in the
// FileGroupKindHierarchyConfig
func ValidateInheritance(config FileGroupKindHierarchyConfig) error {
	if config.GroupKind() == kinds.ResourceQuota().GroupKind() {
		return errIfNotAllowed(config, resourceQuotaModes)
	}
	return errIfNotAllowed(config, otherTypesModes)
}

// errIfNotAllowed returns an error if the kindHierarchyConfig has an inheritance mode which is not allowed for that Kind.
func errIfNotAllowed(config FileGroupKindHierarchyConfig, allowed map[v1alpha1.HierarchyModeType]bool) error {
	if allowed[config.HierarchyMode] {
		return nil
	}
	return vet.IllegalHierarchyModeError{
		HierarchyConfig: config,
		HierarchyMode:   config.HierarchyMode,
		Allowed:         allowed,
	}
}
