package transform

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	sel "github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nodeContext keeps track of objects during the tree traversal for purposes of inheriting values.
type nodeContext struct {
	nodeType  node.Type              // the type of node being processed
	nodePath  cmpath.Path            // the node's path, used for annotating inherited objects
	inherited []*ast.NamespaceObject // the objects that are inherited from the node.
}

// InheritanceSpec defines the spec for inherited resources.
type InheritanceSpec struct {
	Mode v1.HierarchyModeType
}

// InheritanceVisitor aggregates hierarchical quota.
type InheritanceVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
	// groupKinds contains the set of GroupKind that will be targeted during the inheritance transform.
	inheritanceSpecs map[schema.GroupKind]*InheritanceSpec
	// treeContext is a stack that tracks ancestry and inherited objects during the tree traversal.
	treeContext []nodeContext
	// cumulative errors encountered by the visitor
	errs status.MultiError
}

var _ ast.Visitor = &InheritanceVisitor{}

// NewInheritanceVisitor returns a new InheritanceVisitor for the given GroupKind
func NewInheritanceVisitor(specs map[schema.GroupKind]*InheritanceSpec) *InheritanceVisitor {
	iv := &InheritanceVisitor{
		Copying:          visitor.NewCopying(),
		inheritanceSpecs: specs,
	}
	iv.SetImpl(iv)
	return iv
}

// Error implements Visitor
func (v *InheritanceVisitor) Error() status.MultiError {
	return nil
}

// VisitTreeNode implements Visitor
//
// Copies inherited objects into their Namespaces. Otherwise mutating the object later in any
// individual object modifies all copies in other Namespaces.
func (v *InheritanceVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	v.treeContext = append(v.treeContext, nodeContext{
		nodeType: n.Type,
		nodePath: n.Path,
	})
	newNode := v.Copying.VisitTreeNode(n)
	v.treeContext = v.treeContext[:len(v.treeContext)-1]
	if n.Type == node.Namespace {
		for _, ctx := range v.treeContext {
			for _, inherited := range ctx.inherited {
				isApplicable, err := sel.IsConfigApplicableToNamespace(n.Labels, inherited.MetaObject())
				v.errs = status.Append(v.errs, err)
				if err != nil {
					continue
				}
				if isApplicable {
					newNode.Objects = append(newNode.Objects, inherited.DeepCopy())
				}
			}
		}
	}
	return newNode
}

// VisitObject implements Visitor
func (v *InheritanceVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	context := &v.treeContext[len(v.treeContext)-1]
	gk := o.GroupVersionKind().GroupKind()
	if context.nodeType == node.AbstractNamespace {
		spec, found := v.inheritanceSpecs[gk]
		// If the mode is explicitly set to default or is omitted from the HierarchyConfig, use inherit.
		if !found || (found && spec.Mode == v1.HierarchyModeInherit) {
			context.inherited = append(context.inherited, o)
			return nil
		}
	}
	return o
}
