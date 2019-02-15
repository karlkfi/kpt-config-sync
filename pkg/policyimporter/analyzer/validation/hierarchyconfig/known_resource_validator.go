package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/discovery"
)

// KnownResourceValidator validates that HierarchyConfig resources can be looked up using Discovery API.
type KnownResourceValidator struct {
	*visitor.ValidatorBase
	apiInfo *discovery.APIInfo
}

// NewKnownResourceValidator returns a new KnownResourceValidator.
func NewKnownResourceValidator() ast.Visitor {
	return visitor.NewValidator(&KnownResourceValidator{})
}

// ValidateRoot implement ast.Visitor.
func (k *KnownResourceValidator) ValidateRoot(r *ast.Root) error {
	k.apiInfo = discovery.GetAPIInfo(r)
	return nil
}

// ValidateSystemObject implements Visitor.
func (k *KnownResourceValidator) ValidateSystemObject(o *ast.SystemObject) error {
	switch h := o.Object.(type) {
	case *v1alpha1.HierarchyConfig:
		for _, gkc := range NewFileHierarchyConfig(h, o.Relative).flatten() {
			gk := gkc.GroupKind()
			if !k.apiInfo.GroupKindExists(gk) {
				return vet.UnknownResourceInHierarchyConfigError{HierarchyConfig: gkc}
			}
		}
	}
	return nil
}
