package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// FileObjectValidator validates the local state of a single ast.FileObject
type FileObjectValidator struct {
	// ValidateFn returns an error if it finds any problems with the state of the passed FileObject.
	ValidateFn func(object ast.FileObject) error
}

// Validate validates each ast.FileObject individually
func (v FileObjectValidator) Validate(fileObjects []ast.FileObject, errorBuilder *status.ErrorBuilder) {
	for _, fileObject := range fileObjects {
		errorBuilder.Add(v.ValidateFn(fileObject))
	}
}
