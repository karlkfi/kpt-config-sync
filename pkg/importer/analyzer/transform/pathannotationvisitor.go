package transform

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
)

// PathAnnotationVisitor sets "configmanagement.gke.io/source-path" annotation on objects.
type pathAnnotationVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
}

var _ ast.Visitor = &pathAnnotationVisitor{}

// NewPathAnnotationVisitor returns a new PathAnnotationVisitor
func NewPathAnnotationVisitor() ast.Visitor {
	v := &pathAnnotationVisitor{
		Copying: visitor.NewCopying(),
	}
	v.SetImpl(v)
	return v
}

// VisitClusterObject implements Visitor
func (v *pathAnnotationVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	newObject := v.Copying.VisitClusterObject(o)
	core.SetAnnotation(newObject, v1.SourcePathAnnotationKey, o.SlashPath())
	return newObject
}

// VisitObject implements Visitor
func (v *pathAnnotationVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	newObject := v.Copying.VisitObject(o)
	core.SetAnnotation(newObject, v1.SourcePathAnnotationKey, o.SlashPath())
	return newObject
}
