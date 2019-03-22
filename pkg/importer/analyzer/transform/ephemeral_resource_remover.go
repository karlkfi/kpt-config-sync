package transform

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
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

// VisitRoot implements ast.Visitor.
func (r *EphemeralResourceRemover) VisitRoot(n *ast.Root) *ast.Root {
	var nonEphemeralSystem []*ast.SystemObject
	for _, o := range n.SystemObjects {
		if !IsEphemeral(o.GroupVersionKind()) {
			nonEphemeralSystem = append(nonEphemeralSystem, o)
		}
	}
	n.SystemObjects = nonEphemeralSystem

	var nonEphemeralCluserRegistry []*ast.ClusterRegistryObject
	for _, o := range n.ClusterRegistryObjects {
		if !IsEphemeral(o.GroupVersionKind()) {
			nonEphemeralCluserRegistry = append(nonEphemeralCluserRegistry, o)
		}
	}
	n.ClusterRegistryObjects = nonEphemeralCluserRegistry

	var nonEphemeralCluster []*ast.ClusterObject
	for _, o := range n.ClusterObjects {
		if !IsEphemeral(o.GroupVersionKind()) {
			nonEphemeralCluster = append(nonEphemeralCluster, o)
		}
	}
	n.ClusterObjects = nonEphemeralCluster

	return r.Base.VisitRoot(n)
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
