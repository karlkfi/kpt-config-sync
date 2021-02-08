package tree

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// SystemBuilderVisitor sets up the system/ directory from a set of objects in the system directory,
// including adding all objects to Root.System.Objects and setting Root.Repo if a repo object is
// defined.
type SystemBuilderVisitor struct {
	objects []ast.FileObject
	errs    status.MultiError
	*visitor.Base
}

// NewSystemBuilderVisitor initializes a SystemBuilderVisitor.
func NewSystemBuilderVisitor(objects []ast.FileObject) *SystemBuilderVisitor {
	v := &SystemBuilderVisitor{
		Base:    visitor.NewBase(),
		objects: objects,
		errs:    nil,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds System to Root if there are any objects to add.
// Also sets repo if one exists.
func (v *SystemBuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	for _, object := range v.objects {
		if object.GroupVersionKind() != kinds.Repo() {
			continue
		}
		s, err := object.Structured()
		if err != nil {
			v.errs = status.Append(v.errs, err)
			continue
		}
		r.Repo = s.(*v1.Repo)
	}

	for _, o := range v.objects {
		r.SystemObjects = append(r.SystemObjects, &ast.SystemObject{FileObject: o})
	}

	return r
}

func (v *SystemBuilderVisitor) Error() status.MultiError {
	return v.errs
}
