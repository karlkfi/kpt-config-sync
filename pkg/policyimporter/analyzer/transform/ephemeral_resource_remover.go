package transform

import (
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func toResourceList(gvk schema.GroupVersionKind, namespaced namespacedKind) *metav1.APIResourceList {
	return &metav1.APIResourceList{
		GroupVersion: gvk.GroupVersion().String(),
		APIResources: []metav1.APIResource{{Kind: gvk.Kind, Namespaced: bool(namespaced)}},
	}
}

// namespacedKind marks whether a Kind is namespaced or not.
type namespacedKind bool

func ephemeralKinds() map[schema.GroupVersionKind]namespacedKind {
	return map[schema.GroupVersionKind]namespacedKind{
		kinds.Namespace():         true,
		kinds.NamespaceSelector(): true,
	}
}

// IsEphemeral returns true if the gvk is used and valiated client-side, but not uploaded to K8S.
func IsEphemeral(gvk schema.GroupVersionKind) bool {
	_, found := ephemeralKinds()[gvk]
	return found
}

// EphemeralResources returns the APIResourceLists of the ephemeral resources.
func EphemeralResources() []*metav1.APIResourceList {
	var result []*metav1.APIResourceList
	for gvk, namespaced := range ephemeralKinds() {
		result = append(result, toResourceList(gvk, namespaced))
	}
	return result
}

// EphemeralResourceRemover removes ephemeral resources from the policy hierarchy.
type EphemeralResourceRemover struct {
	*visitor.Base
}

var _ ast.Visitor = &EphemeralResourceRemover{}

// NewEphemeralResourceRemover initializes an EphemeralResourceRemover.
func NewEphemeralResourceRemover() *EphemeralResourceRemover {
	v := &EphemeralResourceRemover{Base: visitor.NewBase()}
	v.SetImpl(v)
	return v
}

// VisitSystem implements ast.Visitor.
func (r *EphemeralResourceRemover) VisitSystem(n *ast.System) *ast.System {
	var nonEphemeral []*ast.SystemObject
	for _, o := range n.Objects {
		if !IsEphemeral(o.GroupVersionKind()) {
			nonEphemeral = append(nonEphemeral, o)
		}
	}
	n.Objects = nonEphemeral

	return r.Base.VisitSystem(n)
}

// VisitClusterRegistry implements ast.Visitor.
func (r *EphemeralResourceRemover) VisitClusterRegistry(n *ast.ClusterRegistry) *ast.ClusterRegistry {
	var nonEphemeral []*ast.ClusterRegistryObject
	for _, o := range n.Objects {
		if !IsEphemeral(o.GroupVersionKind()) {
			nonEphemeral = append(nonEphemeral, o)
		}
	}
	n.Objects = nonEphemeral

	return r.Base.VisitClusterRegistry(n)
}

// VisitCluster implements ast.Visitor.
func (r *EphemeralResourceRemover) VisitCluster(n *ast.Cluster) *ast.Cluster {
	var nonEphemeral []*ast.ClusterObject
	for _, o := range n.Objects {
		if !IsEphemeral(o.GroupVersionKind()) {
			nonEphemeral = append(nonEphemeral, o)
		}
	}
	n.Objects = nonEphemeral

	return r.Base.VisitCluster(n)
}

// VisitTreeNode implements ast.Visitor.
func (r *EphemeralResourceRemover) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	var nonEphemeral []*ast.NamespaceObject
	for _, o := range n.Objects {
		if !IsEphemeral(o.GroupVersionKind()) {
			nonEphemeral = append(nonEphemeral, o)
		}
	}
	n.Objects = nonEphemeral

	return r.Base.VisitTreeNode(n)
}
