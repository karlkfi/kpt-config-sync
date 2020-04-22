package tree

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
)

// clusterRegistryBuilderVisitor adds all cluster objects to the config root.
type clusterRegistryBuilderVisitor struct {
	objects []ast.FileObject
	*visitor.Base
}

// NewClusterRegistryBuilderVisitor instantiates a clusterRegistryBuilderVisitor with a set of objects to add.
func NewClusterRegistryBuilderVisitor(objects []ast.FileObject) ast.Visitor {
	v := &clusterRegistryBuilderVisitor{
		Base:    visitor.NewBase(),
		objects: objects,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds ClusterRegistry to Root if there are any objects to add.
func (v *clusterRegistryBuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	for _, o := range v.objects {
		r.ClusterRegistryObjects = append(r.ClusterRegistryObjects, &ast.ClusterRegistryObject{FileObject: o})
	}
	return r
}
