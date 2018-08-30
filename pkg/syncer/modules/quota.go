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
// Reviewed by sunilarora

package modules

import (
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	controller_informers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// AggregatedQuota provides aggregation operations for the ResourceQuota resource.
type AggregatedQuota struct {
	limits core_v1.ResourceList
}

// AggregatedQuota implements hierarchy.AggregatedNode
var _ hierarchy.AggregatedNode = &AggregatedQuota{}

// MergeLimits takes two ResourceList objects and performs the union on all specified limits.   Conflicting
// limits will be resolved by taking the lower of the two.
func MergeLimits(lhs, rhs core_v1.ResourceList) core_v1.ResourceList {
	merged := core_v1.ResourceList{}
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

// Aggregated implements hierarchy.AggregatedNode
func (s *AggregatedQuota) Aggregated(node *policyhierarchy_v1.PolicyNode) hierarchy.AggregatedNode {
	if node.Spec.ResourceQuotaV1 != nil {
		return &AggregatedQuota{limits: MergeLimits(node.Spec.ResourceQuotaV1.Spec.Hard, s.limits)}
	}
	return &AggregatedQuota{limits: s.limits}
}

// Generate implements hierarchy.AggregatedNode
func (s *AggregatedQuota) Generate() hierarchy.Instances {
	var instances hierarchy.Instances
	if len(s.limits) == 0 {
		return instances
	}

	instances = append(instances, &core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   resourcequota.ResourceQuotaObjectName,
			Labels: resourcequota.NewNomosQuotaLabels(),
		},
		Spec: core_v1.ResourceQuotaSpec{
			Hard: s.limits,
		},
	})
	return instances
}

// ResourceQuota implements a module for flattening quota.
type ResourceQuota struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ policyhierarchycontroller.Module = &ResourceQuota{}

// NewResourceQuota creates the module.
func NewResourceQuota(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *ResourceQuota {
	return &ResourceQuota{
		client:    client,
		informers: informers,
	}
}

// Name implements policyhierarchycontroller.Module
func (s *ResourceQuota) Name() string {
	return "ResourceQuota"
}

// Equal implements policyhierarchycontroller.Module
func (s *ResourceQuota) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*core_v1.ResourceQuota)
	rhs := rhsObj.(*core_v1.ResourceQuota)
	return resourcequota.ResourceListsEqual(lhs.Spec.Hard, rhs.Spec.Hard)
}

// equalSpec performs equals on runtime.Objects
func (s *ResourceQuota) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// NewAggregatedNode implements policyhierarchycontroller.Module
func (s *ResourceQuota) NewAggregatedNode() hierarchy.AggregatedNode {
	return &AggregatedQuota{}
}

// Instance implements policyhierarchycontroller.Module
func (s *ResourceQuota) Instance() meta_v1.Object {
	return &core_v1.ResourceQuota{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *ResourceQuota) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Core().V1().ResourceQuotas()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *ResourceQuota) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&core_v1.ResourceQuota{},
		core_v1.SchemeGroupVersion,
		s.equalSpec,
		s.client.CoreV1(),
		s.informers.Core().V1().ResourceQuotas().Lister(),
	)
}
