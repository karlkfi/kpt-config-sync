package final

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/final/validate"
)

type finalValidator func(objs []ast.FileObject) status.MultiError

// Validation performs final validation checks against the given FileObjects.
// This should be called after all hydration steps are complete so that it can
// validate the final state of the repo.
func Validation(objs []ast.FileObject) status.MultiError {
	var errs status.MultiError
	// See the note about ordering in raw.Hierarchical().
	validators := []finalValidator{
		validate.DuplicateNames,
		validate.UnmanagedNamespaces,
	}
	for _, validator := range validators {
		errs = status.Append(errs, validator(objs))
	}
	return errs
}
