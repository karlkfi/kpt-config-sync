package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/status"
)

// DisallowedFieldsValidator adapts syntax.DisallowedFieldsValidator logic for non-hierarchical file
// structures.
var DisallowedFieldsValidator = perObjectValidator(func(o ast.FileObject) status.Error {
	return syntax.DisallowFields(o)
})
