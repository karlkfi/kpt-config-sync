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
	"fmt"
	"reflect"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	controller_informers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// AggregatedRoleBinding provides aggregation operations for the RoleBinding resource.
type AggregatedRoleBinding struct {
	namespace      bool
	policyNodeName string
	parent         *AggregatedRoleBinding
	roleBindings   []rbac_v1.RoleBinding
}

// Aggregated implements hierarchy.AggregatedNode
func (s *AggregatedRoleBinding) Aggregated(node *policyhierarchy_v1.PolicyNode) hierarchy.AggregatedNode {
	return &AggregatedRoleBinding{
		namespace:      node.Spec.Type.IsNamespace(),
		policyNodeName: node.Name,
		parent:         s,
		roleBindings:   node.Spec.RoleBindingsV1,
	}
}

// Generate implements hierarchy.AggregatedNode
func (s *AggregatedRoleBinding) Generate() hierarchy.Instances {
	var instances hierarchy.Instances
	for node := s; node != nil; node = node.parent {
		for idx := range node.roleBindings {
			roleBinding := &node.roleBindings[idx]
			var rrb *rbac_v1.RoleBinding
			if node.namespace {
				rrb = roleBinding
			} else {
				rrb = roleBinding.DeepCopy()
				rrb.Name = fmt.Sprintf("%s.%s", node.policyNodeName, roleBinding.Name)
			}
			instances = append(instances, rrb)
		}
	}
	return instances
}

var _ hierarchy.AggregatedNode = &AggregatedRoleBinding{}

// RoleBinding implements a module for flattening roles.
type RoleBinding struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ policyhierarchycontroller.Module = &RoleBinding{}

// NewRoleBinding creates the module.
func NewRoleBinding(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *RoleBinding {
	return &RoleBinding{
		client:    client,
		informers: informers,
	}
}

// Name implements policyhierarchycontroller.Module
func (s *RoleBinding) Name() string {
	return "RoleBinding"
}

func (s *RoleBinding) subjectsEqual(lhs *rbac_v1.RoleBinding, rhs *rbac_v1.RoleBinding) bool {
	if len(lhs.Subjects) == 0 && len(rhs.Subjects) == 0 {
		return true
	}
	return reflect.DeepEqual(lhs.Subjects, rhs.Subjects)
}

// Equal implements policyhierarchycontroller.Module
func (s *RoleBinding) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*rbac_v1.RoleBinding)
	rhs := rhsObj.(*rbac_v1.RoleBinding)
	return reflect.DeepEqual(lhs.RoleRef, rhs.RoleRef) && s.subjectsEqual(lhs, rhs)
}

// equalSpec performs equals on runtime.Objects
func (s *RoleBinding) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// NewAggregatedNode implements policyhierarchycontroller.Module
func (s *RoleBinding) NewAggregatedNode() hierarchy.AggregatedNode {
	return &AggregatedRoleBinding{}
}

// Instance implements policyhierarchycontroller.Module
func (s *RoleBinding) Instance() meta_v1.Object {
	return &rbac_v1.RoleBinding{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *RoleBinding) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Rbac().V1().RoleBindings()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *RoleBinding) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbac_v1.RoleBinding{},
		rbac_v1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().RoleBindings().Lister(),
	)
}
