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
	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	controllerinformers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

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

func (s *RoleBinding) subjectsEqual(lhs *rbacv1.RoleBinding, rhs *rbacv1.RoleBinding) bool {
	if len(lhs.Subjects) == 0 && len(rhs.Subjects) == 0 {
		return true
	}
	return reflect.DeepEqual(lhs.Subjects, rhs.Subjects)
}

// Equal implements policyhierarchycontroller.Module
func (s *RoleBinding) Equal(lhsObj metav1.Object, rhsObj metav1.Object) bool {
	lhs := lhsObj.(*rbacv1.RoleBinding)
	rhs := rhsObj.(*rbacv1.RoleBinding)
	return reflect.DeepEqual(lhs.RoleRef, rhs.RoleRef) && s.subjectsEqual(lhs, rhs)
}

// equalSpec performs equals on runtime.Objects
func (s *RoleBinding) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(metav1.Object), rhsObj.(metav1.Object))
}

// Instances implements policyhierarchycontroller.Module
func (s *RoleBinding) Instances(policyNode *policyhierarchyv1.PolicyNode) []metav1.Object {
	var rbs []metav1.Object
	for _, o := range policyNode.Spec.RoleBindingsV1 {
		rbs = append(rbs, o.DeepCopy())
	}
	return rbs
}

// Instance implements policyhierarchycontroller.Module
func (s *RoleBinding) Instance() metav1.Object {
	return &rbacv1.RoleBinding{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *RoleBinding) InformerProvider() controllerinformers.InformerProvider {
	return s.informers.Rbac().V1().RoleBindings()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *RoleBinding) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbacv1.RoleBinding{},
		rbacv1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().RoleBindings().Lister(),
	)
}
