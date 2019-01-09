package transform

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PathAnnotationVisitor sets "nomos.dev/source-path" annotation on CRDs and native objects.
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
	if newNode.Annotations == nil {
		newNode.Annotations = map[string]string{}
	}
	newNode.Annotations[v1alpha1.SourcePathAnnotationKey] = n.Path
	return newNode
}

// VisitClusterObject implements Visitor
func (v *PathAnnotationVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	newObject := v.Copying.VisitClusterObject(o)
	applyPathAnnotation(newObject.FileObject)
	return newObject
}

// VisitObject implements Visitor
func (v *PathAnnotationVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	newObject := v.Copying.VisitObject(o)
	applyPathAnnotation(newObject.FileObject)
	return newObject
}

// applyPathAnnotation applies path annotation to o.
// dir is a slash-separated path.
func applyPathAnnotation(fo ast.FileObject) {
	metaObj := fo.Object.(metav1.Object)
	a := metaObj.GetAnnotations()
	if a == nil {
		a = map[string]string{}
		metaObj.SetAnnotations(a)
	}
	a[v1alpha1.SourcePathAnnotationKey] = fo.RelativeSlashPath()
}
