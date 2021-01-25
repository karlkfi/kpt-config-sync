package parsed

import (
	"github.com/google/nomos/pkg/status"
)

// ValidatorFunc is a function that visits some portion of a Root and validates
// it.
type ValidatorFunc func(root Root) status.MultiError

// ValidateAllObjects returns a ValidatorFunc that calls the given visitor VisitorFunc
// on all objects in a Root.
func ValidateAllObjects(f VisitorFunc) ValidatorFunc {
	return func(root Root) status.MultiError {
		return root.VisitAllObjects(f)
	}
}

// ValidateClusterObjects returns a ValidatorFunc that calls the given visitor VisitorFunc
// all objects under the cluster/ directory.
func ValidateClusterObjects(f VisitorFunc) ValidatorFunc {
	return func(root Root) status.MultiError {
		return root.VisitClusterObjects(f)
	}
}

// ValidateClusterRegistryObjects returns a ValidatorFunc that calls the given visitor VisitorFunc
// all objects under the clusterregistry/ directory.
func ValidateClusterRegistryObjects(f VisitorFunc) ValidatorFunc {
	return func(root Root) status.MultiError {
		return root.VisitClusterRegistryObjects(f)
	}
}

// ValidateNamespaceObjects returns a ValidatorFunc that calls the given visitor VisitorFunc
// all objects under the namespaces/ directory.
func ValidateNamespaceObjects(f VisitorFunc) ValidatorFunc {
	return func(root Root) status.MultiError {
		return root.VisitNamespaceObjects(f)
	}
}

// ValidateSystemObjects returns a ValidatorFunc that calls the given visitor VisitorFunc
// all objects under the system/ directory.
func ValidateSystemObjects(f VisitorFunc) ValidatorFunc {
	return func(root Root) status.MultiError {
		return root.VisitSystemObjects(f)
	}
}
