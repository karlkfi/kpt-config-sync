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

// Role implements a module for flattening roles.
type Role struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

// NewRole creates the module.
func NewRole(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *Role {
	return &Role{
		client:    client,
		informers: informers,
	}
}

var _ policyhierarchycontroller.Module = &Role{}

// Name implements policyhierarchycontroller.Module
func (s *Role) Name() string {
	return "Role"
}

// Equal implements policyhierarchycontroller.Module
func (s *Role) Equal(lhsObj metav1.Object, rhsObj metav1.Object) bool {
	lhs := lhsObj.(*rbacv1.Role)
	rhs := rhsObj.(*rbacv1.Role)
	if len(lhs.Rules) == 0 && len(rhs.Rules) == 0 {
		return true
	}
	return reflect.DeepEqual(lhs.Rules, rhs.Rules)
}

// equalSpec performs equals on runtime.Objects
func (s *Role) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(metav1.Object), rhsObj.(metav1.Object))
}

// Instances implements policyhierarchycontroller.Module
func (s *Role) Instances(policyNode *policyhierarchyv1.PolicyNode) []metav1.Object {
	var rs []metav1.Object
	for i := range policyNode.Spec.RolesV1 {
		rs = append(rs, &policyNode.Spec.RolesV1[i])
	}
	return rs
}

// Instance implements policyhierarchycontroller.Module
func (s *Role) Instance() metav1.Object {
	return &rbacv1.Role{}
}

// InformerProvider implements policyhierarchycontroller.Module
func (s *Role) InformerProvider() controllerinformers.InformerProvider {
	return s.informers.Rbac().V1().Roles()
}

// ActionSpec implements policyhierarchycontroller.Module
func (s *Role) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbacv1.Role{},
		rbacv1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().Roles().Lister(),
	)
}
