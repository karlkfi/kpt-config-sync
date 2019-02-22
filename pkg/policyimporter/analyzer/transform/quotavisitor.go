/*
Copyright 2018 The Nomos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transform

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/resourcequota"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QuotaVisitor aggregates hierarchical quota.  Aggregation is performed by taking the union
// of all defined quotas along the ancestry.  If a conflict between quotas is encountered, for
// example, two nodes define CPU quota, the lower value is used.
type QuotaVisitor struct {
	*visitor.Copying // The copying base class
	// For adding cluster scoped Hierarchical Quota.
	ctx   *quotaContext // The context list for the hierarchy
	hNode *v1.HierarchicalQuotaNode
}

var _ ast.Visitor = &QuotaVisitor{}

// quotaContext keeps track of the ancestry's quota policies.
type quotaContext struct {
	prev  *quotaContext         // previous context
	quota *corev1.ResourceQuota // ResourceQuota from directory
	hNode *v1.HierarchicalQuotaNode
}

// merge takes two resource quota objects and produces a merged output that represents the union
// of the two policies with common fields resolved by taking the minimum.
func merge(lhs, rhs *corev1.ResourceQuota) *corev1.ResourceQuota {
	if rhs == nil {
		return lhs
	}
	if lhs == nil {
		return rhs
	}
	return &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resourcequota.ResourceQuotaObjectName,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: MergeLimits(lhs.Spec.Hard, rhs.Spec.Hard),
		},
	}
}

// aggregated returns the hierarchical aggregated qutoa for the given context
func (qc *quotaContext) aggregated() *corev1.ResourceQuota {
	if qc.prev == nil {
		return qc.quota
	}
	return merge(qc.prev.aggregated(), qc.quota)
}

// NewQuotaVisitor creates a new hierarchical aggregating visitor for quota.
func NewQuotaVisitor() *QuotaVisitor {
	qv := &QuotaVisitor{Copying: visitor.NewCopying()}
	qv.SetImpl(qv)
	return qv
}

// Error implements Visitor
func (v *QuotaVisitor) Error() error {
	return nil
}

// VisitRoot implements Visitor.
func (v *QuotaVisitor) VisitRoot(c *ast.Root) *ast.Root {
	v.hNode = &v1.HierarchicalQuotaNode{}
	newRoot := v.Copying.VisitRoot(c)

	h := &v1.HierarchicalQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "HierarchicalQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resourcequota.ResourceQuotaHierarchyName,
		},
		Spec: v1.HierarchicalQuotaSpec{
			Hierarchy: *v.hNode,
		},
	}

	// Add completed hierarchical quota tree to cluster objects.
	if newRoot.Cluster == nil {
		newRoot.Cluster = &ast.Cluster{}
	}
	newRoot.Cluster.Objects = append(newRoot.Cluster.Objects, &ast.ClusterObject{
		FileObject: ast.FileObject{Object: h},
	})
	return newRoot
}

// VisitTreeNode implements Visitor
func (v *QuotaVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	// create/push context
	context := &quotaContext{
		prev:  v.ctx,
		hNode: &v1.HierarchicalQuotaNode{},
	}

	v.ctx = context
	newNode := v.Copying.VisitTreeNode(n)

	if n.Type == node.Namespace {
		context.hNode.Name = n.Name()
		context.hNode.Type = v1.HierarchyNodeNamespace
	} else {
		context.hNode.Name = n.RelativeSlashPath()
		context.hNode.Type = v1.HierarchyNodeAbstractNamespace
	}

	if (n.Type == node.AbstractNamespace && context.quota != nil) || (n.Type == node.Namespace) {
		if quota := context.aggregated(); quota != nil {
			if n.Type == node.Namespace {
				quota = quota.DeepCopy()
				quota.Labels = resourcequota.NewNomosQuotaLabels()
			}
			context.hNode.ResourceQuotaV1 = quota
			newNode.Objects = append(newNode.Objects, &ast.NamespaceObject{FileObject: ast.FileObject{Object: quota}})
		}
	}
	if context.prev != nil {
		context.prev.hNode.Children = append(context.prev.hNode.Children, *context.hNode)
	} else {
		// context.prev == nil implies root TreeNode so copy the root HierarchicalQuotaNode to the other context
		v.hNode = context.hNode
	}
	v.ctx = context.prev
	return newNode
}

// VisitObject implements Visitor, this should only be visited if the objectset
// is of type ResourceQuota.
func (v *QuotaVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	if kinds.ResourceQuota() == o.GetObjectKind().GroupVersionKind() {
		quota := *o.FileObject.Object.(*corev1.ResourceQuota)
		quota.Name = resourcequota.ResourceQuotaObjectName
		v.ctx.quota = merge(&quota, v.ctx.quota)
		return nil
	}
	return o
}

// MergeLimits takes two ResourceList objects and performs the union on all specified limits.   Conflicting
// limits will be resolved by taking the lower of the two.
func MergeLimits(lhs, rhs corev1.ResourceList) corev1.ResourceList {
	merged := corev1.ResourceList{}
	for k, v := range lhs {
		merged[k] = v
	}
	for k, v := range rhs {
		if limit, exists := merged[k]; !exists || (v.Cmp(limit) < 0) {
			merged[k] = v
		}
	}
	return merged
}
