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
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	controllerinformers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

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
func (s *ResourceQuota) Equal(lhsObj metav1.Object, rhsObj metav1.Object) bool {
	lhs := lhsObj.(*corev1.ResourceQuota)
	rhs := rhsObj.(*corev1.ResourceQuota)
	return resourcequota.ResourceListsEqual(lhs.Spec.Hard, rhs.Spec.Hard)
}

// equalSpec performs equals on runtime.Objects
func (s *ResourceQuota) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(metav1.Object), rhsObj.(metav1.Object))
}

// Instances implements policyhierarchycontroller.Module
func (s *ResourceQuota) Instances(policyNode *v1.PolicyNode) []metav1.Object {
	if policyNode.Spec.ResourceQuotaV1 != nil {
		return []metav1.Object{policyNode.Spec.ResourceQuotaV1}
	}
	return nil
}

// Instance implements policyhierarchycontroller.Module
func (s *ResourceQuota) Instance() metav1.Object {
	return &corev1.ResourceQuota{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *ResourceQuota) InformerProvider() controllerinformers.InformerProvider {
	return s.informers.Core().V1().ResourceQuotas()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *ResourceQuota) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&corev1.ResourceQuota{},
		corev1.SchemeGroupVersion,
		s.equalSpec,
		s.client.CoreV1(),
		s.informers.Core().V1().ResourceQuotas().Lister(),
	)
}
