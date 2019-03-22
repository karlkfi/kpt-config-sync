package object

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
)

// Predicate is a function that accepts a FileObject and returns true or false.
type Predicate func(object ast.FileObject) bool

// True returns a Predicate that always returns true.
func True() Predicate {
	return func(_ ast.FileObject) bool {
		return true
	}
}

// False returns a Predicate that always returns false.
func False() Predicate {
	return func(object ast.FileObject) bool {
		return false
	}
}

// All returns a Predicate which returns true iff all passed predicates return true.
func All(ps ...Predicate) Predicate {
	return func(object ast.FileObject) bool {
		for _, p := range ps {
			if !p(object) {
				return false
			}
		}
		return true
	}
}

// Any returns a Predicate which returns true iff any of the passed Predicates returns true.
func Any(ps ...Predicate) Predicate {
	return func(object ast.FileObject) bool {
		for _, p := range ps {
			if p(object) {
				return true
			}
		}
		return false
	}
}

// Not returns a Predicate which returns true iff the underlying Predicate would return false.
func Not(p Predicate) Predicate {
	return func(object ast.FileObject) bool {
		return !p(object)
	}
}

// Filter returns a list of the passed FileObjects for which p returns false.
// Preserves the relative order of passed objects.
func Filter(objects []ast.FileObject, p Predicate) []ast.FileObject {
	var result []ast.FileObject
	for _, object := range objects {
		if p(object) {
			continue
		}
		result = append(result, object)
	}
	return result
}
