package transform

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/object"
)

// PathAnnotationVisitor sets "configmanagement.gke.io/source-path" annotation on CRDs and native objects.
type PathAnnotationVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
}

var _ ast.Visitor = &PathAnnotationVisitor{}

// NewPathAnnotationVisitor returns a new PathAnnotationVisitor
func NewPathAnnotationVisitor() *PathAnnotationVisitor {
	v := &PathAnnotationVisitor{
		Copying: visitor.NewCopying(),
	}
	v.SetImpl(v)
	return v
}

// VisitTreeNode implements Visitor
func (v *PathAnnotationVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	newNode := v.Copying.VisitTreeNode(n)
	object.SetAnnotation(newNode, v1.SourcePathAnnotationKey, n.SlashPath())
	return newNode
}

// VisitClusterObject implements Visitor
func (v *PathAnnotationVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	newObject := v.Copying.VisitClusterObject(o)
	object.SetAnnotation(newObject.MetaObject(), v1.SourcePathAnnotationKey, o.SlashPath())
	return newObject
}

// VisitObject implements Visitor
func (v *PathAnnotationVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	newObject := v.Copying.VisitObject(o)
	object.SetAnnotation(newObject.MetaObject(), v1.SourcePathAnnotationKey, o.SlashPath())
	return newObject
}
