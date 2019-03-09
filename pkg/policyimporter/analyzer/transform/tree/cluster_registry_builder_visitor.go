package tree

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// ClusterRegistryBuilderVisitor adds all cluster objects to the policy root.
type ClusterRegistryBuilderVisitor struct {
	objects []ast.FileObject
	*visitor.Base
}

// NewClusterRegistryBuilderVisitor instantiates a ClusterRegistryBuilderVisitor with a set of objects to add.
func NewClusterRegistryBuilderVisitor(objects []ast.FileObject) *ClusterRegistryBuilderVisitor {
	v := &ClusterRegistryBuilderVisitor{
		Base:    visitor.NewBase(),
		objects: objects,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds ClusterRegistry to Root if there are any objects to add.
func (v *ClusterRegistryBuilderVisitor) VisitRoot(r *ast.Root) *ast.Root {
	for _, o := range v.objects {
		r.ClusterRegistryObjects = append(r.ClusterRegistryObjects, &ast.ClusterRegistryObject{FileObject: o})
	}
	return r
}
