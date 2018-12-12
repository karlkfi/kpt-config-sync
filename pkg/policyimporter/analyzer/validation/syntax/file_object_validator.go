package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/util/multierror"
)

// FileObjectValidator validates the local state of a single ast.FileObject
type FileObjectValidator struct {
	validate func(object ast.FileObject) error
}

// Validate validates each ast.FileObject individually
func (v FileObjectValidator) Validate(fileObjects []ast.FileObject, errorBuilder *multierror.Builder) {
	for _, fileObject := range fileObjects {
		errorBuilder.Add(v.validate(fileObject))
	}
}
