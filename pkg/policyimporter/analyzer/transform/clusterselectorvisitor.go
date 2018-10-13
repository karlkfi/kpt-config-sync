package transform

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// ClusterSelectorVisitor filters out tree elements and objects that do not match
// the provided cluster selector.
type ClusterSelectorVisitor struct {
	*visitor.Copying
	// Unset until VisitRoot returns.
	selectors *ClusterSelectors
}

var _ ast.Visitor = (*ClusterSelectorVisitor)(nil)

// NewClusterSelectorVisitor creates a new visitor that filters the tree based on a
// cluster selector (passed in as part of the root).
func NewClusterSelectorVisitor() *ClusterSelectorVisitor {
	v := &ClusterSelectorVisitor{
		Copying: visitor.NewCopying(),
	}
	v.SetImpl(v)
	return v
}

// VisitRoot implements ast.Visitor.
func (v *ClusterSelectorVisitor) VisitRoot(r *ast.Root) ast.Node {
	v.selectors = GetClusterSelectors(r)
	return v.Copying.VisitRoot(r)
}

// VisitTreeNode prunes the tree node (and by extension all objects in and
// nodes below it) if it doesn't match the active cluster selector.
func (v *ClusterSelectorVisitor) VisitTreeNode(n *ast.TreeNode) ast.Node {
	glog.V(6).Infof("VisitTreeNode(%v): enter: %+v", n.Path, *n)
	defer glog.V(6).Infof("VisitTreeNode(%v): exit", n.Path)
	if !v.selectors.Matches(n) {
		glog.V(5).Infof("VisitTreeNode(%v): omit", n.Path)
		// Omit this tree node and children.
		return nil
	}
	return v.Copying.VisitTreeNode(n)

}

// VisitObject prunes a namespace object if it doesn't match the active cluster selector.
// If the containing tree node doesn't match, however, the object does won't ever be visited
// and will be filtered out as result.
func (v *ClusterSelectorVisitor) VisitObject(o *ast.NamespaceObject) ast.Node {
	glog.V(6).Infof("VisitObject(): enter")
	defer glog.V(6).Infof("VisitObject(): exit")
	if !v.selectors.Matches(o.ToMeta()) {
		glog.V(5).Infof("VisitObject(): omit")
		// Omit this object.
		return nil
	}
	return o
}

// VisitClusterObject prunes a cluster scoped object if it doesn't match the
// active cluster selector.
func (v *ClusterSelectorVisitor) VisitClusterObject(o *ast.ClusterObject) ast.Node {
	glog.V(6).Infof("VisitClusterObject(): enter")
	defer glog.V(6).Infof("VisitClusterObject(): exit")
	if !v.selectors.Matches(o.ToMeta()) {
		glog.V(5).Infof("VisitClusterObject(): omit")
		// Omit this object.
		return nil
	}
	return o
}
