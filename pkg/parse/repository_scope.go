package parse

import (
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OptionsForScope returns new Options that have been updated for the given
// Scope.
func OptionsForScope(options validate.Options, scope declared.Scope) validate.Options {
	if scope == declared.RootReconciler {
		options.DefaultNamespace = metav1.NamespaceDefault
		options.IsNamespaceReconciler = false
	} else {
		options.DefaultNamespace = string(scope)
		options.IsNamespaceReconciler = true
		options.Visitors = append(options.Visitors, repositoryScopeVisitor(scope))
	}
	return options
}

// repositoryScopeVisitor ensures all objects in a Namespace Repo are either
// 1) The Namespace for the scope, or
// 2) Namespace-scoped objects that define metadata.namespace matching the scope, or
//      omit metadata.namespace.
func repositoryScopeVisitor(scope declared.Scope) validate.VisitorFunc {
	return func(objs []ast.FileObject) ([]ast.FileObject, status.MultiError) {
		var errs status.MultiError
		for _, obj := range objs {
			// By this point we've validated that there are no cluster-scoped objects
			// in this repo.
			switch obj.GetNamespace() {
			case string(scope):
				// This is what we want, so ignore.
			case "":
				// Missing metadata.namespace, so set it to be the one for this Repo.
				// Otherwise this will invalidly default to the "default" Namespace.
				obj.SetNamespace(string(scope))
			default:
				// There's an object declaring an invalid metadata.namespace, so this is
				// an error.
				errs = status.Append(errs, BadScopeErr(obj, scope))
			}
		}
		return objs, errs
	}
}

// BadScopeErr reports that the passed resource declares a Namespace for a
// different Namespace repository.
func BadScopeErr(resource client.Object, want declared.Scope) status.ResourceError {
	return nonhierarchical.BadScopeErrBuilder.
		Sprintf("Resources in the %q repo must either omit metadata.namespace or declare metadata.namespace=%q", want, want).
		BuildWithResources(resource)
}
