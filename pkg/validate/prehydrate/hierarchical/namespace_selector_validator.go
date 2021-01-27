package hierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
)

// NamespaceSelectorValidator returns a visitor which verifies that
// NamespaceSelector objects are only located in abstract namespaces.
func NamespaceSelectorValidator() parsed.ValidatorFunc {
	return parsed.ValidateNamespaceObjects(validateNamespaceSelectors)
}

func validateNamespaceSelectors(objs []ast.FileObject) status.MultiError {
	var selectors []id.Resource
	var isNamespace bool
	for _, obj := range objs {
		switch obj.GroupVersionKind() {
		case kinds.Namespace():
			isNamespace = true
		case kinds.NamespaceSelector():
			selectors = append(selectors, obj)
		}
	}
	if isNamespace && len(selectors) > 0 {
		return syntax.IllegalKindInNamespacesError(selectors...)
	}
	return nil
}
