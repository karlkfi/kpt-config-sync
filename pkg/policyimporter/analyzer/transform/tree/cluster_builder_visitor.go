package tree

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
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
	if (r.Cluster == nil) && (len(v.objects) > 0) {
		r.Cluster = &ast.Cluster{}
	}

	return v.Base.VisitRoot(r)
}

// VisitCluster adds the cluster objects to Cluster.
func (v *ClusterBuilderVisitor) VisitCluster(c *ast.Cluster) *ast.Cluster {
	for _, o := range v.objects {
		c.Objects = append(c.Objects, &ast.ClusterObject{FileObject: o})
	}
	return c
}
