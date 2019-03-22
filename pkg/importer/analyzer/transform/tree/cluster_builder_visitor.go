package tree

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
)

// ClusterBuilderVisitor adds all cluster objects to the policy root.
type ClusterBuilderVisitor struct {
	objects []ast.FileObject
	*visitor.Base
}

// NewClusterBuilderVisitor instantiates a ClusterBuilderVisitor with a set of objects to add.
func NewClusterBuilderVisitor(objects []ast.FileObject) *ClusterBuilderVisitor {
	v := &ClusterBuilderVisitor{
		Base:    visitor.NewBase(),
		objects: objects,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds Cluster to Root if there are any objects to add.
func (v *ClusterBuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	for _, o := range v.objects {
		r.ClusterObjects = append(r.ClusterObjects, &ast.ClusterObject{FileObject: o})
	}
	return r
}
