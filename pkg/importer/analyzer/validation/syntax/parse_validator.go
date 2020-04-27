package syntax

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var structuredKinds = map[schema.GroupKind]bool{
	kinds.Cluster().GroupKind():           true,
	kinds.ClusterSelector().GroupKind():   true,
	kinds.HierarchyConfig().GroupKind():   true,
	kinds.NamespaceSelector().GroupKind(): true,
	kinds.Repo().GroupKind():              true,
}

// MustBeStructured returns true if the importer logic requires the given GroupKind
// to be parsed into a structured object.
func MustBeStructured(gvk schema.GroupVersionKind) bool {
	return structuredKinds[gvk.GroupKind()]
}

// NewParseValidator returns a ValidatorVisitor which ensures required types are actually
// instantiated rather than read into unstructured.Unstructureds.
//
// Objects which are read into *unstructured.Unstructured instead of the go type (if one is
// available) mean the config is improperly formatted. Note that this condition only applies
// to certain resource kinds (mostly configmanagement) which are required by the importer's
// logic (eg NamespaceSelector to determine which namespaces an object should sync to).
func NewParseValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		_, isUnstructured := o.Object.(*unstructured.Unstructured)
		if MustBeStructured(o.GroupVersionKind()) {
			if isUnstructured {
				return ObjectParseError(&o)
			}
		} else if !isUnstructured {
			glog.Warningf("Resource should have been parsed as unstructured: %s %s/%s", o.GroupVersionKind(), o.GetNamespace(), o.GetName())
		}
		return nil
	})
}

// ObjectParseErrorCode is the code for ObjectParseError.
const ObjectParseErrorCode = "1006"

var objectParseError = status.NewErrorBuilder(ObjectParseErrorCode)

// ObjectParseError reports that an object of known type did not match its definition, and so it was
// read in as an *unstructured.Unstructured.
func ObjectParseError(resource id.Resource) status.Error {
	return objectParseError.
		Sprintf("The following config could not be parsed as a %v:", resource.GroupVersionKind()).
		BuildWithResources(resource)
}
