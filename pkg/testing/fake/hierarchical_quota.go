package fake

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/resourcequota"
	corev1 "k8s.io/api/core/v1"
)

// HierarchicalQuotaMutator modifies a HierarchicalQuota.
type HierarchicalQuotaMutator func(hc *v1.HierarchicalQuota)

// HierarchicalQuotaRoot sets the root of the HierarchicalQuota hierarchy.
func HierarchicalQuotaRoot(root v1.HierarchicalQuotaNode) HierarchicalQuotaMutator {
	return func(hc *v1.HierarchicalQuota) {
		hc.Spec.Hierarchy = root
	}
}

// HierarchicalQuotaNode initializes a HierarchicalQuotaNode.
func HierarchicalQuotaNode(name string, nodeType v1.HierarchyNodeType, rq *corev1.ResourceQuota, children ...v1.HierarchicalQuotaNode) v1.HierarchicalQuotaNode {
	return v1.HierarchicalQuotaNode{
		Name:            name,
		Type:            nodeType,
		ResourceQuotaV1: rq,
		Children:        children,
	}
}

// HierarchicalQuotaObject initializes a HierarchicalQuota.
func HierarchicalQuotaObject(opts ...HierarchicalQuotaMutator) *v1.HierarchicalQuota {
	result := &v1.HierarchicalQuota{TypeMeta: toTypeMeta(kinds.HierarchicalQuota())}
	defaultMutate(result)
	mutate(result, object.Name(resourcequota.ResourceQuotaHierarchyName))
	for _, opt := range opts {
		opt(result)
	}

	return result
}
