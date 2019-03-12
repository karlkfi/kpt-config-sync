package mutate

import (
	"github.com/google/nomos/pkg/cloner/filter"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// Mutator modifies an ast.FileObject.
type Mutator func(object *ast.FileObject)

// Apply modifies all passed objects.
func (m Mutator) Apply(objects []ast.FileObject) {
	if m == nil {
		return
	}
	for i := range objects {
		m(&objects[i])
	}
}

// If returns a Mutator which only runs on objects where f returns true.
func (m Mutator) If(p filter.Predicate) Mutator {
	if m == nil {
		return nil
	}
	return func(object *ast.FileObject) {
		if p(*object) {
			m(object)
		}
	}
}

// ApplyAll mutates the list of objects with the Mutator.
func ApplyAll(objects []ast.FileObject, ms ...Mutator) {
	for _, m := range ms {
		m.Apply(objects)
	}
}
