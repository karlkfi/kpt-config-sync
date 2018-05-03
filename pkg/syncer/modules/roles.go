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

// AggregatedRole provides aggregation operations for the Role resource.
type AggregatedRole struct {
	roles []rbac_v1.Role
}

// Aggregated implements hierarchy.AggregatedNode
func (s *AggregatedRole) Aggregated(node *policyhierarchy_v1.PolicyNode) hierarchy.AggregatedNode {
	if !node.Spec.Type.IsNamespace() {
		return &AggregatedRole{}
	}

	return &AggregatedRole{roles: node.Spec.RolesV1}
}

// Generate implements hierarchy.AggregatedNode
func (s *AggregatedRole) Generate() hierarchy.Instances {
	var instances hierarchy.Instances
	for _, role := range s.roles {
		instances = append(instances, role.DeepCopy())
	}
	return instances
}

var _ hierarchy.AggregatedNode = &AggregatedRole{}

// RoleModule implements a module for flattening roles.
type RoleModule struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

// NewRoleModule creates the module.
func NewRoleModule(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *RoleModule {
	return &RoleModule{
		client:    client,
		informers: informers,
	}
}

var _ policyhierarchycontroller.Module = &RoleModule{}

// Name implements policyhierarchycontroller.Module
func (s *RoleModule) Name() string {
	return "Role"
}

// Equal implements policyhierarchycontroller.Module
func (s *RoleModule) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*rbac_v1.Role)
	rhs := rhsObj.(*rbac_v1.Role)
	if len(lhs.Rules) == 0 && len(rhs.Rules) == 0 {
		return true
	}
	return reflect.DeepEqual(lhs.Rules, rhs.Rules)
}

// equalSpec performs equals on runtime.Objects
func (s *RoleModule) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// NewAggregatedNode implements policyhierarchycontroller.Module
func (s *RoleModule) NewAggregatedNode() hierarchy.AggregatedNode {
	return &AggregatedRole{}
}

// Instance implements policyhierarchycontroller.Module
func (s *RoleModule) Instance() meta_v1.Object {
	return &rbac_v1.Role{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *RoleModule) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Rbac().V1().Roles()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *RoleModule) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbac_v1.Role{},
		rbac_v1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().Roles().Lister(),
	)
}
