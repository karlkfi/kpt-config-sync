package syntax

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var requiredTypedStructs = map[schema.GroupVersionKind]bool{
	kinds.Cluster():           true,
	kinds.HierarchyConfig():   true,
	kinds.NamespaceSelector(): true,
	kinds.Repo():              true,
}

// NewParseValidator returns a ValidatorVisitor which ensures required types are actually
// instantiated rather than read into unstructured.Unstructureds.
//
// Objects which are read into *unstructured.Unstructured instead of the go type (if one is
// available) mean the config is improperly formatted. Note that this condition only applies
// to Nomos CRDs, as go type definitions for other types (e.g. Application) are not available
// to the parser.
func NewParseValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		if _, ok := o.Object.(*unstructured.Unstructured); ok {
			if requiredTypedStructs[o.GroupVersionKind()] {
				return status.From(vet.ObjectParseError(&o))
			}
		}
		return nil
	})
}
