package syntax

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type disallowSystemObjectsValidator struct {
	visitor.ValidatorBase
}

// ValidateClusterRegistryObject implements visitor.Validator.
func (v *disallowSystemObjectsValidator) ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) status.MultiError {
	if IsSystemOnly(o.GroupVersionKind()) {
		return IllegalSystemResourcePlacementError(o)
	}
	return nil
}

// ValidateClusterObject implements visitor.Validator.
func (v *disallowSystemObjectsValidator) ValidateClusterObject(o *ast.ClusterObject) status.MultiError {
	if IsSystemOnly(o.GroupVersionKind()) {
		return IllegalSystemResourcePlacementError(o)
	}
	return nil
}

// ValidateObject implements visitor.Validator.
func (v *disallowSystemObjectsValidator) ValidateObject(o *ast.NamespaceObject) status.MultiError {
	if IsSystemOnly(o.GroupVersionKind()) {
		return IllegalSystemResourcePlacementError(o)
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

// IllegalSystemResourcePlacementErrorCode is the error code for IllegalSystemResourcePlacementError
const IllegalSystemResourcePlacementErrorCode = "1033"

var illegalSystemResourcePlacementError = status.NewErrorBuilder(IllegalSystemResourcePlacementErrorCode)

// IllegalSystemResourcePlacementError reports that a configmanagement.gke.io object has been defined outside of system/
func IllegalSystemResourcePlacementError(resource id.Resource) status.Error {
	return illegalSystemResourcePlacementError.
		Sprintf("A config of the below kind MUST NOT be declared outside %[1]s/:",
			repo.SystemDir).
		BuildWithResources(resource)
}
