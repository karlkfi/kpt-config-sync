package syntax

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewNamespaceKindValidator returns a Validator that ensures only the allowed set of Kinds appear
// in various Namespaces.
func NewNamespaceKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewTreeNodeValidator(func(n *ast.TreeNode) status.MultiError {
		if n.Type != node.Namespace && n.Type != node.AbstractNamespace {
			// Exit early on node types not handled by this Visitor.
			return nil
		}
		if glog.V(6) {
			// Useful for debugging.
			glog.Infof("node: %v", spew.Sdump(n))
		}

		var eb status.MultiError
		for _, object := range n.Objects {
			// Common forbidden objects.  Checked by GVK as not all objects have
			// a go type backing that we need for typesafe checks.
			if e := checkForbiddenGVK(object, commonForbidden); e != nil {
				eb = status.Append(eb, e)
			}
			// Objects only forbidden in Namespaces.
			if n.Type == node.Namespace {
				if e := checkForbiddenGVK(object, forbiddenInNamespace); e != nil {
					eb = status.Append(eb, illegalObject(object))
				}
			}
		}
		return eb
	})
}

func illegalObject(r *ast.NamespaceObject) vet.IllegalKindInNamespacesError {
	return vet.IllegalKindInNamespacesError{Resource: r}
}

var (
	commonForbidden = map[schema.GroupKind]bool{
		kinds.ConfigManagement().GroupKind(): true,
	}
	forbiddenInNamespace = map[schema.GroupKind]bool{
		kinds.NamespaceSelector().GroupKind(): true,
	}
)

// checkForbiddenGVK checks for disallowed objects that do not have a go type
// backing.
func checkForbiddenGVK(o *ast.NamespaceObject, disallowed map[schema.GroupKind]bool) error {
	gk := o.GetObjectKind().GroupVersionKind().GroupKind()
	if glog.V(7) {
		glog.Infof("gk: %+v, compare: %+v", gk, kinds.ConfigManagement().GroupKind())
	}
	if disallowed[gk] {
		return illegalObject(o)
	}
	return nil
}
