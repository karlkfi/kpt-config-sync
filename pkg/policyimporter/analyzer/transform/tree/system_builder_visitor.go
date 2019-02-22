package tree

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// SystemBuilderVisitor sets up the system/ directory from a set of objects in the system directory,
// including adding all objects to Root.System.Objects and setting Root.Repo if a repo object is
// defined.
type SystemBuilderVisitor struct {
	objects []ast.FileObject
	*visitor.Base
}

// NewSystemBuilderVisitor initializes a SystemBuilderVisitor.
func NewSystemBuilderVisitor(objects []ast.FileObject) *SystemBuilderVisitor {
	v := &SystemBuilderVisitor{
		Base:    visitor.NewBase(),
		objects: objects,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds System to Root if there are any objects to add.
// Also sets repo if one exists.
func (v *SystemBuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	for _, object := range v.objects {
		switch o := object.Object.(type) {
		case *v1.Repo:
			r.Repo = o
		}
	}

	if (r.System == nil) && (len(v.objects) > 0) {
		r.System = &ast.System{}
	}

	return v.Base.VisitRoot(r)
}

// VisitSystem adds the System objects to System.
func (v *SystemBuilderVisitor) VisitSystem(c *ast.System) *ast.System {
	for _, o := range v.objects {
		c.Objects = append(c.Objects, &ast.SystemObject{FileObject: o})
	}
	return c
}
