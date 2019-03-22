package object

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
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
func (m Mutator) If(p Predicate) Mutator {
	if m == nil {
		return nil
	}
	return func(object *ast.FileObject) {
		if p(*object) {
			m(object)
		}
	}
}

// Mutate returns a Mutator that applies all underlying mutations.
func Mutate(ms ...Mutator) Mutator {
	return func(object *ast.FileObject) {
		for _, m := range ms {
			if m == nil {
				continue
			}
			m(object)
		}
	}
}

// Path replaces the path with the provided slash-delimited path from nomos root.
func Path(path string) Mutator {
	return func(o *ast.FileObject) {
		o.Path = cmpath.FromSlash(path)
	}
}
