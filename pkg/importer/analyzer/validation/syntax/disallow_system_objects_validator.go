package syntax

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type disallowSystemObjectsValidator struct {
	visitor.ValidatorBase
}

// ValidateClusterRegistryObject implements visitor.Validator.
func (v *disallowSystemObjectsValidator) ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) *status.MultiError {
	if IsSystemOnly(o.GroupVersionKind()) {
		return status.From(vet.IllegalSystemResourcePlacementError{Resource: o})
	}
	return nil
}

// ValidateClusterObject implements visitor.Validator.
func (v *disallowSystemObjectsValidator) ValidateClusterObject(o *ast.ClusterObject) *status.MultiError {
	if IsSystemOnly(o.GroupVersionKind()) {
		return status.From(vet.IllegalSystemResourcePlacementError{Resource: o})
	}
	return nil
}

// ValidateObject implements visitor.Validator.
func (v *disallowSystemObjectsValidator) ValidateObject(o *ast.NamespaceObject) *status.MultiError {
	if IsSystemOnly(o.GroupVersionKind()) {
		return status.From(vet.IllegalSystemResourcePlacementError{Resource: o})
	}
	return nil
}

// NewDisallowSystemObjectsValidator validates that the resources which may appear in system/ and nowhere
// else only appear in system/.
func NewDisallowSystemObjectsValidator() *visitor.ValidatorVisitor {
	return visitor.NewValidator(&disallowSystemObjectsValidator{})
}

// IsSystemOnly returns true if the GVK is only allowed in the system/ directory.
// It returns true iff the object is allowed in system/, but no other directories.
func IsSystemOnly(gvk schema.GroupVersionKind) bool {
	switch gvk {
	case kinds.Repo(), kinds.HierarchyConfig():
		return true
	default:
		return false
	}
}
