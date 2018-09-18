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
	"reflect"

	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/clusterpolicycontroller"
	controllerinformers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// ClusterRoleBindings implements a module for comparing
// clusterrolebindings and generating actions to update them.
type ClusterRoleBindings struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ clusterpolicycontroller.Module = &ClusterRoleBindings{}

// NewClusterRoleBindings creates the module.
func NewClusterRoleBindings(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *ClusterRoleBindings {
	return &ClusterRoleBindings{
		client:    client,
		informers: informers,
	}
}

// Name implements clusterpolicycontroller.Module.
func (s *ClusterRoleBindings) Name() string {
	return "ClusterRoleBindings"
}

func (s *ClusterRoleBindings) subjectsEqual(lhs *rbacv1.ClusterRoleBinding, rhs *rbacv1.ClusterRoleBinding) bool {
	if len(lhs.Subjects) == 0 && len(rhs.Subjects) == 0 {
		return true
	}
	return reflect.DeepEqual(lhs.Subjects, rhs.Subjects)
}

// Equal implements clusterpolicycontroller.Module.
func (s *ClusterRoleBindings) Equal(lhsObj metav1.Object, rhsObj metav1.Object) bool {
	lhs := lhsObj.(*rbacv1.ClusterRoleBinding)
	rhs := rhsObj.(*rbacv1.ClusterRoleBinding)

	if lhs == nil || rhs == nil {
		return lhs == rhs
	}

	return reflect.DeepEqual(lhs.RoleRef, rhs.RoleRef) && s.subjectsEqual(lhs, rhs)
}

// equalSpec performs equals on runtime.Objects
func (s *ClusterRoleBindings) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(metav1.Object), rhsObj.(metav1.Object))
}

// InformerProvider implements clusterpolicycontroller.Module
func (s *ClusterRoleBindings) InformerProvider() controllerinformers.InformerProvider {
	return s.informers.Rbac().V1().ClusterRoleBindings()
}

// Instance implements clusterpolicycontroller.Module
func (s *ClusterRoleBindings) Instance() metav1.Object {
	return &rbacv1.ClusterRoleBinding{}
}

// Extract implements clusterpolicycontroller.Module
func (s *ClusterRoleBindings) Extract(clusterPolicy *policyhierarchyv1.ClusterPolicy) []metav1.Object {
	var roles []runtime.Object
	for _, r := range clusterPolicy.Spec.ClusterRoleBindingsV1 {
		roles = append(roles, r.DeepCopy())
	}
	return object.RuntimeToMeta(roles)
}

// ActionSpec implements clusterpolicycontroller.Module
func (s *ClusterRoleBindings) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbacv1.ClusterRoleBinding{},
		corev1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().ClusterRoleBindings().Lister(),
	)
}
