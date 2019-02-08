package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/discovery"
)

// NewKnownResourceValidator returns a Visitor that adds errors for resources that cannot be looked up using Discovery API.
func NewKnownResourceValidator(apiInfo *discovery.APIInfo) *visitor.ValidatorVisitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) error {
		switch h := o.Object.(type) {
		case *v1alpha1.HierarchyConfig:
			for _, gkc := range NewFileHierarchyConfig(h, o.Relative).flatten() {
				gk := gkc.GroupKind()
				if !apiInfo.GroupKindExists(gk) {
					return vet.UnknownResourceInHierarchyConfigError{HierarchyConfig: gkc}
				}
			}
		}
		return nil
	})
}
