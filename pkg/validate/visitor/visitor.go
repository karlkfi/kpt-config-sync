package visitor

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// Func is a function that visits a collection of FileObjects.
type Func func(objects []ast.FileObject) status.MultiError

type objectFunc func(object ast.FileObject) status.Error

// PerObjectFunc returns a Func that calls the given function on each FileObject
// in a collection.
func PerObjectFunc(f objectFunc) Func {
	return func(objects []ast.FileObject) status.MultiError {
		var err status.MultiError
		for _, o := range objects {
			err = status.Append(err, f(o))
		}
		return err
	}
}
