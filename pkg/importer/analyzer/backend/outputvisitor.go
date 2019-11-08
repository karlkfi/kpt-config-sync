package backend

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
)

// OutputVisitor converts the AST into NamespaceConfig and ClusterConfig objects.
type OutputVisitor struct {
	*visitor.Base
	allConfigs *namespaceconfig.AllConfigs
}

var _ ast.Visitor = &OutputVisitor{}

// NewOutputVisitor creates a new output visitor.
func NewOutputVisitor() *OutputVisitor {
	v := &OutputVisitor{Base: visitor.NewBase()}
	v.SetImpl(v)
	return v
}

// AllConfigs returns the AllConfigs object created by the visitor.
func (v *OutputVisitor) AllConfigs() *namespaceconfig.AllConfigs {
	return v.allConfigs
}

// VisitRoot implements Visitor.
func (v *OutputVisitor) VisitRoot(g *ast.Root) *ast.Root {
	v.allConfigs = namespaceconfig.NewAllConfigs(g.ImportToken, g.LoadTime)
	v.Base.VisitRoot(g)
	return nil
}

// VisitSystemObject implements Visitor.
func (v *OutputVisitor) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	if sync, ok := o.Object.(*v1.Sync); ok {
		v.allConfigs.AddSync(*sync)
	}
	return o
}

// VisitTreeNode implements Visitor.
func (v *OutputVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	if n.Type == node.Namespace {
		// Only emit NamespaceConfigs for leaf nodes.
		v.allConfigs.AddNamespaceConfig(n.Name(), n.Annotations, n.Labels)
		// Just loop inside here rather than using VisitObject becuase this is easier to follow.
		for _, o := range n.Objects {
			o.SetNamespace(n.Name())
			v.allConfigs.AddNamespaceResource(n.Name(), o.Object)
		}
		// Namespace has no children, so no need to process further.
	} else {
		// Visit children of the Abstract Namespace
		v.Base.VisitTreeNode(n)
	}
	return nil
}

// VisitClusterObject implements Visitor.
func (v *OutputVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	v.allConfigs.AddClusterResource(o.Object)
	return nil
}

// Error implements Visitor.
func (v *OutputVisitor) Error() status.MultiError {
	return nil
}

// RequiresValidState returns true because we don't want to output configs if there are problems.
func (v *OutputVisitor) RequiresValidState() bool {
	return true
}
