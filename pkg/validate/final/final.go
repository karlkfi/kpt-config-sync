package final

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/final/validate"
)

type finalValidator func(objs []ast.FileObject) status.MultiError

// Hierarchical performs final validation checks for a structured hierarchical
// repo against the given FileObjects. This should be called after all hydration
// steps are complete so that it can validate the final state of the repo.
func Hierarchical(objs []ast.FileObject) status.MultiError {
	var errs status.MultiError
	validators := []finalValidator{
		validate.DuplicateNames,
	}
	for _, validator := range validators {
		errs = status.Append(errs, validator(objs))
	}
	return errs
}

// Unstructured performs final validation checks for an unstructured repo
// against the given FileObjects. This should be called after all hydration
// steps are complete so that it can validate the final state of the repo.
func Unstructured(objs []ast.FileObject) status.MultiError {
	var errs status.MultiError
	validators := []finalValidator{
		validate.DuplicateNames,
	}
	for _, validator := range validators {
		errs = status.Append(errs, validator(objs))
	}
	return errs
}
