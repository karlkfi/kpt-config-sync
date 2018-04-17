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

// ClusterRoleBindingsModule implements a module for comparing
// clusterrolebindings and generating actions to update them.
type ClusterRoleBindingsModule struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ clusterpolicycontroller.Module = &ClusterRoleBindingsModule{}

// NewClusterRoleBindingsModule creates the module.
func NewClusterRoleBindingsModule(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *ClusterRoleBindingsModule {
	return &ClusterRoleBindingsModule{
		client:    client,
		informers: informers,
	}
}

// Name implements clusterpolicycontroller.Module.
func (s *ClusterRoleBindingsModule) Name() string {
	return "ClusterRoleBindings"
}

func (s *ClusterRoleBindingsModule) subjectsEqual(lhs *rbac_v1.ClusterRoleBinding, rhs *rbac_v1.ClusterRoleBinding) bool {
	if len(lhs.Subjects) == 0 && len(rhs.Subjects) == 0 {
		return true
	}
	return reflect.DeepEqual(lhs.Subjects, rhs.Subjects)
}

// Equal implements clusterpolicycontroller.Module.
func (s *ClusterRoleBindingsModule) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*rbac_v1.ClusterRoleBinding)
	rhs := rhsObj.(*rbac_v1.ClusterRoleBinding)
	if lhs == nil && rhs == nil {
		return true
	}
	if lhs == nil || rhs == nil {
		// Both can be nil, but not one or the other.
		return false
	}
	return reflect.DeepEqual(lhs.RoleRef, rhs.RoleRef) && s.subjectsEqual(lhs, rhs)
}

// equalSpec performs equals on runtime.Objects
func (s *ClusterRoleBindingsModule) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// InformerProvider implements clusterpolicycontroller.Module
func (s *ClusterRoleBindingsModule) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Rbac().V1().ClusterRoleBindings()
}

// Instance implements clusterpolicycontroller.Module
func (s *ClusterRoleBindingsModule) Instance() meta_v1.Object {
	return &rbac_v1.ClusterRoleBinding{}
}

// Extract implements clusterpolicycontroller.Module
func (s *ClusterRoleBindingsModule) Extract(clusterPolicy *policyhierarchy_v1.ClusterPolicy) []meta_v1.Object {
	var roles []runtime.Object
	for _, r := range clusterPolicy.Spec.Policies.ClusterRoleBindingsV1 {
		roles = append(roles, &r)
	}
	return object.RuntimeToMeta(roles)
}

// ActionSpec implements clusterpolicycontroller.Module
func (s *ClusterRoleBindingsModule) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbac_v1.ClusterRoleBinding{},
		core_v1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().ClusterRoleBindings().Lister(),
	)
}
