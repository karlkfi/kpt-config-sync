package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// DuplicateNameValidator forbids declaring conlficting resources.
var DuplicateNameValidator = validator{
	validate: func(objects []ast.FileObject) status.MultiError {
		resources := make([]id.Resource, len(objects))
		for i, o := range objects {
			obj := o // Use intermediate object since taking the reference of a loop variable is bad.
			resources[i] = &obj
		}
		return metadata.CheckDuplicates(resources)
	},
}
