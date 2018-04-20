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
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/clusterpolicycontroller"
	controller_informers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	core_v1 "k8s.io/api/core/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// ClusterRolesModule implements a module for comparing clusterroles and
// generating actions to update them.
type ClusterRolesModule struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ clusterpolicycontroller.Module = &ClusterRolesModule{}

// NewClusterRolesModule creates the module.
func NewClusterRolesModule(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *ClusterRolesModule {
	return &ClusterRolesModule{
		client:    client,
		informers: informers,
	}
}

// Name implements clusterpolicycontroller.Module.
func (s *ClusterRolesModule) Name() string {
	return "ClusterRoles"
}

// Equal implements clusterpolicycontroller.Module.
func (s *ClusterRolesModule) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*rbac_v1.ClusterRole)
	rhs := rhsObj.(*rbac_v1.ClusterRole)

	if lhs == nil && rhs == nil {
		return true
	}
	if lhs == nil || rhs == nil {
		// Both can be nil, but not one or the other.
		return false
	}
	if lhs.AggregationRule != nil || rhs.AggregationRule != nil {
		return reflect.DeepEqual(lhs.AggregationRule, rhs.AggregationRule)
	}
	return reflect.DeepEqual(lhs.Rules, rhs.Rules)
}

// equalSpec performs equals on runtime.Objects
func (s *ClusterRolesModule) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// InformerProvider implements clusterpolicycontroller.Module
func (s *ClusterRolesModule) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Rbac().V1().ClusterRoles()
}

// Instance implements clusterpolicycontroller.Module
func (s *ClusterRolesModule) Instance() meta_v1.Object {
	return &rbac_v1.ClusterRole{}
}

// Extract implements clusterpolicycontroller.Module
func (s *ClusterRolesModule) Extract(clusterPolicy *policyhierarchy_v1.ClusterPolicy) []meta_v1.Object {
	var roles []runtime.Object
	for _, r := range clusterPolicy.Spec.Policies.ClusterRolesV1 {
		roles = append(roles, &r)
	}
	return object.RuntimeToMeta(roles)
}

// ActionSpec implements clusterpolicycontroller.Module
func (s *ClusterRolesModule) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbac_v1.ClusterRole{},
		core_v1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().ClusterRoles().Lister(),
	)
}
