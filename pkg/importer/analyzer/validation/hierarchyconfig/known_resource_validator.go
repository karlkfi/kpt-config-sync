package hierarchyconfig

import (
	"fmt"

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
	scoper discovery.Scoper
}

// NewKnownResourceValidator returns a new KnownResourceValidator.
func NewKnownResourceValidator() ast.Visitor {
	return visitor.NewValidator(&KnownResourceValidator{})
}

// ValidateRoot implement ast.Visitor.
func (k *KnownResourceValidator) ValidateRoot(r *ast.Root) status.MultiError {
	var err error
	k.scoper, err = discovery.GetScoper(r)
	return status.From(err)
}

// ValidateSystemObject implements Visitor.
func (k *KnownResourceValidator) ValidateSystemObject(o *ast.SystemObject) status.MultiError {
	var errs []error
	switch h := o.Object.(type) {
	case *v1.HierarchyConfig:
		for _, gkc := range NewFileHierarchyConfig(h, o).flatten() {
			if err := k.validateGroupKind(gkc); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return status.From(errs...)
}

// validateGroupKind validates a group kind for both existing in the API discovery as well as
// being at namespace scope.
func (k *KnownResourceValidator) validateGroupKind(gkc FileGroupKindHierarchyConfig) error {
	switch scope := k.scoper.GetScope(gkc.GroupKind()); scope {
	case discovery.UnknownScope:
		return status.From(vet.UnknownResourceInHierarchyConfigError(gkc))
	case discovery.NamespaceScope:
		return nil
	case discovery.ClusterScope:
		return status.From(vet.ClusterScopedResourceInHierarchyConfigError(gkc, scope))
	default:
		panic(fmt.Sprintf("programmer error: case %s should not occur", scope))
	}
}
