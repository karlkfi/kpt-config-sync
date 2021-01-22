package validate

import (
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/visitor"
)

// Root represents a collection of declared configs grouped by major directory.
type Root interface {
	// VisitAllObjects calls the given visitor.Func on all objects in the Root.
	VisitAllObjects(visitor.Func) status.MultiError
	// VisitClusterObjects calls the given visitor.Func on all objects under the
	// cluster/ directory.
	VisitClusterObjects(visitor.Func) status.MultiError
	// VisitClusterRegistryObjects calls the given visitor.Func on all objects
	// under the clusterregistry/ directory.
	VisitClusterRegistryObjects(visitor.Func) status.MultiError
	// VisitNamespaceObjects calls the given visitor.Func on all objects under the
	// namespaces/ directory.
	VisitNamespaceObjects(visitor.Func) status.MultiError
	// VisitSystemObjects calls the given visitor.Func on all objects under the
	// system/ directory.
	VisitSystemObjects(visitor.Func) status.MultiError
}
