package transform

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
)

// UnarySync transforms Sync objects so they only contain one group version.
type UnarySync struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
	syncs []*v1alpha1.Sync
}

// NewUnarySync returns a new UnarySync transform.
func NewUnarySync() *UnarySync {
	v := &UnarySync{
		Copying: visitor.NewCopying(),
	}
	v.Copying.SetImpl(v)
	return v
}

// VisitClusterRegistry implements Visitor
func (v *UnarySync) VisitClusterRegistry(c *ast.ClusterRegistry) *ast.ClusterRegistry {
	return c
}

// VisitCluster implements Visitor
func (v *UnarySync) VisitCluster(c *ast.Cluster) *ast.Cluster {
	return c
}

// VisitTreeNode implements Visitor
func (v *UnarySync) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	return n
}

// VisitSystem implements Visitor
func (v *UnarySync) VisitSystem(c *ast.System) *ast.System {
	sys := v.Copying.VisitSystem(c)
	for _, sync := range v.syncs {
		for _, s := range v.expand(sync) {
			sys.Objects = append(sys.Objects, &ast.SystemObject{FileObject: ast.FileObject{Object: s}})
		}
	}
	return sys
}

// VisitSystemObject implements Visitor
func (v *UnarySync) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	if s, ok := o.FileObject.Object.(*v1alpha1.Sync); ok {
		v.syncs = append(v.syncs, s)
		return nil
	}
	return o
}

// expand transforms a Sync into one or more Sync objects that contain only one group, kind.
func (v *UnarySync) expand(s *v1alpha1.Sync) []*v1alpha1.Sync {
	var syncs []*v1alpha1.Sync
	for _, groupInfo := range s.Spec.Groups {
		for _, kindInfo := range groupInfo.Kinds {
			sync := &v1alpha1.Sync{
				TypeMeta:   s.TypeMeta,
				ObjectMeta: s.ObjectMeta,
				Spec: v1alpha1.SyncSpec{
					Groups: []v1alpha1.SyncGroup{
						{
							Group: groupInfo.Group,
							Kinds: []v1alpha1.SyncKind{
								{
									Kind:          kindInfo.Kind,
									HierarchyMode: kindInfo.HierarchyMode,
									Versions:      kindInfo.Versions,
								},
							},
						},
					},
				},
			}
			if groupInfo.Group == "" {
				sync.Name = strings.ToLower(kindInfo.Kind)
			} else {
				sync.Name = fmt.Sprintf("%s.%s", strings.ToLower(kindInfo.Kind), groupInfo.Group)
			}
			syncs = append(syncs, sync)
		}
	}
	return syncs
}
