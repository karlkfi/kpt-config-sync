package parsed

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// VisitorFunc is a function that visits a collection of FileObjects.
type VisitorFunc func(objects []ast.FileObject) status.MultiError

type objectFunc func(object ast.FileObject) status.Error

// PerObjectVisitor returns a VisitorFunc that calls the given function on each
// FileObject in a collection.
func PerObjectVisitor(f objectFunc) VisitorFunc {
	return func(objects []ast.FileObject) status.MultiError {
		var err status.MultiError
		for _, o := range objects {
			err = status.Append(err, f(o))
		}
		return err
	}
}
