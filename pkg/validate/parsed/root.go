package parsed

import (
	"github.com/google/nomos/pkg/status"
)

// Root represents a collection of declared configs grouped by major directory.
type Root interface {
	// VisitAllObjects calls the given visitor.VisitorFunc on all objects in the Root.
	VisitAllObjects(VisitorFunc) status.MultiError
	// VisitClusterObjects calls the given visitor.VisitorFunc on all objects under the
	// cluster/ directory.
	VisitClusterObjects(VisitorFunc) status.MultiError
	// VisitClusterRegistryObjects calls the given visitor.VisitorFunc on all objects
	// under the clusterregistry/ directory.
	VisitClusterRegistryObjects(VisitorFunc) status.MultiError
	// VisitNamespaceObjects calls the given visitor.VisitorFunc on all objects under the
	// namespaces/ directory.
	VisitNamespaceObjects(VisitorFunc) status.MultiError
	// VisitSystemObjects calls the given visitor.VisitorFunc on all objects under the
	// system/ directory.
	VisitSystemObjects(VisitorFunc) status.MultiError
}
