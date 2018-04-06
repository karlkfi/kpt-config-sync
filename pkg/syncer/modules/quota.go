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

// Aggregated implements hierarchy.AggregatedNode
func (s *AggregatedQuota) Aggregated(node *policyhierarchy_v1.PolicyNode) hierarchy.AggregatedNode {
	limits := core_v1.ResourceList{}
	if node.Spec.Policies.ResourceQuotaV1 != nil {
		for k, v := range node.Spec.Policies.ResourceQuotaV1.Spec.Hard {
			limits[k] = v
		}
	}
	for k, aggregatedLimit := range s.limits {
		if limit, exists := limits[k]; !exists || (exists && aggregatedLimit.Cmp(limit) < 0) {
			limits[k] = aggregatedLimit
		}
	}
	return &AggregatedQuota{limits: limits}
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

// ResourceQuotaModule implements a module for flattening quota.
type ResourceQuotaModule struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ policyhierarchycontroller.Module = &ResourceQuotaModule{}

// NewResourceQuotaModule creates the module.
func NewResourceQuotaModule(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *ResourceQuotaModule {
	return &ResourceQuotaModule{
		client:    client,
		informers: informers,
	}
}

// Name implements policyhierarchycontroller.Module
func (s *ResourceQuotaModule) Name() string {
	return "ResourceQuota"
}

// Equal implements policyhierarchycontroller.Module
func (s *ResourceQuotaModule) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*core_v1.ResourceQuota)
	rhs := rhsObj.(*core_v1.ResourceQuota)
	return resourcequota.ResourceListsEqual(lhs.Spec.Hard, rhs.Spec.Hard)
}

// equalSpec performs equals on runtime.Objects
func (s *ResourceQuotaModule) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// NewAggregatedNode implements policyhierarchycontroller.Module
func (s *ResourceQuotaModule) NewAggregatedNode() hierarchy.AggregatedNode {
	return &AggregatedQuota{}
}

// Instance implements policyhierarchycontroller.Module
func (s *ResourceQuotaModule) Instance() meta_v1.Object {
	return &core_v1.ResourceQuota{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *ResourceQuotaModule) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Core().V1().ResourceQuotas()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *ResourceQuotaModule) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&core_v1.ResourceQuota{},
		core_v1.SchemeGroupVersion,
		s.equalSpec,
		s.client.CoreV1(),
		s.informers.Core().V1().ResourceQuotas().Lister(),
	)
}
