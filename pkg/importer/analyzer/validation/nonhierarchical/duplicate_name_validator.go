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
		return metadata.CheckDuplicates(resources, NameCollisionError)
	},
}

// NameCollisionError reports that multiple objects of the same Kind and Namespace have the same Name.
// For non-hierarchical repositories.
func NameCollisionError(resources ...id.Resource) status.Error {
	return metadata.NameCollisionErrorBuilder.WithResources(resources...).New(
		"No two configs may declare the same API Group, kind, metadata.name, and metadata.namespace",
	)
}
