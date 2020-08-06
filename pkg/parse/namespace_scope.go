package parse

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// namespaceScopeVisitor ensures all objects in a Namespace Repo are either
// 1) The Namespace for the scope, or
// 2) Namespace-scoped objects that define metadata.namespace matching the scope, or
//      omit metadata.namespace.
func namespaceScopeVisitor(scope string) nonhierarchical.Validator {
	return nonhierarchical.PerObjectValidator(func(o ast.FileObject) status.Error {
		// By this point we've validated that there are no cluster-scoped objects
		// in this repo.
		switch o.GetNamespace() {
		case scope:
			// This is what we want, so ignore.
		case "":
			// Missing metadata.namespace, so set it to be the one for this Repo.
			// Otherwise this will invalidly default to the "default" Namespace.
			o.SetNamespace(scope)
		default:
			// There's an object declaring an invalid metadata.namespace, so this is
			// an error.
			return badScopeErr(o, scope)
		}
		return nil
	})
}

var badScopeErrBuilder = status.NewErrorBuilder("1058")

func badScopeErr(resource id.Resource, want string) status.ResourceError {
	return badScopeErrBuilder.
		Sprintf("Resources in the %q repo must either omit metadata.namespace or declare metadata.namespace=%q", want, want).
		BuildWithResources(resource)
}
