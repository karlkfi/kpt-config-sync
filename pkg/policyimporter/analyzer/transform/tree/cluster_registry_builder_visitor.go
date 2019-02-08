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
	if (r.ClusterRegistry == nil) && (len(v.objects) > 0) {
		r.ClusterRegistry = &ast.ClusterRegistry{}
	}

	return v.Base.VisitRoot(r)
}

// VisitClusterRegistry adds the cluster objects to ClusterRegistry.
func (v *ClusterRegistryBuilderVisitor) VisitClusterRegistry(c *ast.ClusterRegistry) *ast.ClusterRegistry {
	for _, o := range v.objects {
		c.Objects = append(c.Objects, &ast.ClusterRegistryObject{FileObject: o})
	}
	return c
}
