package hierarchyconfig

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
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
func (k *KnownResourceValidator) ValidateRoot(r *ast.Root) status.MultiError {
	k.apiInfo = discovery.GetAPIInfo(r)
	return nil
}

// ValidateSystemObject implements Visitor.
func (k *KnownResourceValidator) ValidateSystemObject(o *ast.SystemObject) status.MultiError {
	switch h := o.Object.(type) {
	case *v1.HierarchyConfig:
		for _, gkc := range NewFileHierarchyConfig(h, o).flatten() {
			gk := gkc.GroupKind()
			if !k.apiInfo.GroupKindExists(gk) {
				return status.From(vet.UnknownResourceInHierarchyConfigError{HierarchyConfig: gkc})
			}
		}
	}
	return nil
}
