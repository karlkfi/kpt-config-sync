package transform

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

// PathAnnotationVisitor sets "configmanagement.gke.io/source-path" annotation on objects.
type pathAnnotationVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying

	// We prefix the annotation paths with the policy dir for clarity.
	policyDir cmpath.Relative
}

var _ ast.Visitor = &pathAnnotationVisitor{}

// NewPathAnnotationVisitor returns a new PathAnnotationVisitor
func NewPathAnnotationVisitor(policyDir cmpath.Relative) ast.Visitor {
	v := &pathAnnotationVisitor{
		Copying:   visitor.NewCopying(),
		policyDir: policyDir,
	}
	v.SetImpl(v)
	return v
}

// VisitClusterObject implements Visitor
func (v *pathAnnotationVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	newObject := v.Copying.VisitClusterObject(o)
	core.SetAnnotation(newObject, v1.SourcePathAnnotationKey, v.policyDir.Join(o.Relative).SlashPath())
	return newObject
}

// VisitObject implements Visitor
func (v *pathAnnotationVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	newObject := v.Copying.VisitObject(o)
	core.SetAnnotation(newObject, v1.SourcePathAnnotationKey, v.policyDir.Join(o.Relative).SlashPath())
	return newObject
}
