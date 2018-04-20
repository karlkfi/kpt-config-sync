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
	roleBindings []rbac_v1.RoleBinding
}

// Aggregated implements hierarchy.AggregatedNode
func (s *AggregatedRoleBinding) Aggregated(node *policyhierarchy_v1.PolicyNode) hierarchy.AggregatedNode {
	roleBindings := make([]rbac_v1.RoleBinding, len(s.roleBindings)+len(node.Spec.Policies.RoleBindingsV1))
	copy(roleBindings[0:len(s.roleBindings)], s.roleBindings)
	copy(roleBindings[len(s.roleBindings):], node.Spec.Policies.RoleBindingsV1)
	return &AggregatedRoleBinding{roleBindings: roleBindings}
}

// Generate implements hierarchy.AggregatedNode
func (s *AggregatedRoleBinding) Generate() hierarchy.Instances {
	var instances hierarchy.Instances
	for _, roleBinding := range s.roleBindings {
		instances = append(instances, roleBinding.DeepCopy())
	}
	return instances
}

var _ hierarchy.AggregatedNode = &AggregatedRoleBinding{}

// RoleBindingModule implements a module for flattening roles.
type RoleBindingModule struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ policyhierarchycontroller.Module = &RoleBindingModule{}

// NewRoleBindingModule creates the module.
func NewRoleBindingModule(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *RoleBindingModule {
	return &RoleBindingModule{
		client:    client,
		informers: informers,
	}
}

// Name implements policyhierarchycontroller.Module
func (s *RoleBindingModule) Name() string {
	return "RoleBinding"
}

func (s *RoleBindingModule) subjectsEqual(lhs *rbac_v1.RoleBinding, rhs *rbac_v1.RoleBinding) bool {
	if len(lhs.Subjects) == 0 && len(rhs.Subjects) == 0 {
		return true
	}
	return reflect.DeepEqual(lhs.Subjects, rhs.Subjects)
}

// Equal implements policyhierarchycontroller.Module
func (s *RoleBindingModule) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*rbac_v1.RoleBinding)
	rhs := rhsObj.(*rbac_v1.RoleBinding)
	return reflect.DeepEqual(lhs.RoleRef, rhs.RoleRef) && s.subjectsEqual(lhs, rhs)
}

// equalSpec performs equals on runtime.Objects
func (s *RoleBindingModule) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// NewAggregatedNode implements policyhierarchycontroller.Module
func (s *RoleBindingModule) NewAggregatedNode() hierarchy.AggregatedNode {
	return &AggregatedRoleBinding{}
}

// Instance implements policyhierarchycontroller.Module
func (s *RoleBindingModule) Instance() meta_v1.Object {
	return &rbac_v1.RoleBinding{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *RoleBindingModule) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Rbac().V1().RoleBindings()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *RoleBindingModule) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbac_v1.RoleBinding{},
		rbac_v1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().RoleBindings().Lister(),
	)
}
