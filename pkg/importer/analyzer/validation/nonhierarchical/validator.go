package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// Validator implementors validate a set of non-hierarchical configuration.
type Validator interface {
	// Validate returns a MultiError if the passed FileObjects fail validation.
	Validate([]ast.FileObject) status.MultiError
}

type validator struct {
	validate func([]ast.FileObject) status.MultiError
}

// Validate implements Validator.
func (v validator) Validate(objects []ast.FileObject) status.MultiError {
	return v.validate(objects)
}

var _ Validator = validator{}

func perObjectValidator(fn func(o ast.FileObject) status.Error) Validator {
	return validator{
		validate: func(objects []ast.FileObject) status.MultiError {
			var err status.MultiError
			for _, o := range objects {
				err = status.Append(err, fn(o))
			}
			return err
		},
	}
}
