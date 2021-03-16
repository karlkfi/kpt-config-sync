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
