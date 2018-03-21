/*
Copyright 2017 The Nomos Authors.
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

package actions

import (
	"reflect"

	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	listers_rbac_v1 "k8s.io/client-go/listers/rbac/v1"
)

// RoleBindingResource is an implementation of ResourceInterface for rolebindings.
type RoleBindingResource struct {
	client kubernetes.Interface
	lister listers_rbac_v1.RoleBindingLister
}

// NewRoleBindingResource creates a role binding resource from the kubernetes client objects.
func NewRoleBindingResource(
	client kubernetes.Interface,
	lister listers_rbac_v1.RoleBindingLister) *RoleBindingResource {
	return &RoleBindingResource{
		client: client,
		lister: lister,
	}
}

// RoleResource implements ResourceInterface
var _ ResourceInterface = &RoleBindingResource{}

// Create implements ResourceInterface
func (s RoleBindingResource) Create(obj interface{}) (interface{}, error) {
	roleBinding := obj.(*rbac_v1.RoleBinding)
	return s.client.RbacV1().RoleBindings(roleBinding.Namespace).Create(roleBinding)
}

// Delete implements ResourceInterface
func (s RoleBindingResource) Delete(obj interface{}) error {
	roleBinding := obj.(*rbac_v1.RoleBinding)
	return s.client.RbacV1().RoleBindings(roleBinding.Namespace).Delete(
		roleBinding.Name, &meta_v1.DeleteOptions{})
}

// Get implements ResourceInterface
func (s RoleBindingResource) Get(obj interface{}) (interface{}, error) {
	roleBinding := obj.(*rbac_v1.RoleBinding)
	return s.lister.RoleBindings(roleBinding.Namespace).Get(roleBinding.Name)
}

// Update implements ResourceInterface
func (s RoleBindingResource) Update(oldObj interface{}, newObj interface{}) (interface{}, error) {
	newRoleBinding := newObj.(*rbac_v1.RoleBinding)
	newRoleBinding.ResourceVersion = oldObj.(*rbac_v1.RoleBinding).ResourceVersion
	return s.client.RbacV1().RoleBindings(newRoleBinding.Namespace).Update(newRoleBinding)
}

// Type implements ResourceInterface
func (s RoleBindingResource) Type() string {
	return "rolebinding"
}

// Name implements ResourceInterface
func (s RoleBindingResource) Name(obj interface{}) string {
	return obj.(*rbac_v1.RoleBinding).Name
}

// Namespace implements ResourceInterface
func (s RoleBindingResource) Namespace(obj interface{}) string {
	return obj.(*rbac_v1.RoleBinding).Namespace
}

// Equal implements ResourceInterface
func (s RoleBindingResource) Equal(lhsObj interface{}, rhsObj interface{}) bool {
	lhs := lhsObj.(*rbac_v1.RoleBinding)
	rhs := rhsObj.(*rbac_v1.RoleBinding)
	if MetaEquals(lhs.ObjectMeta, rhs.ObjectMeta) &&
		reflect.DeepEqual(lhs.Subjects, rhs.Subjects) &&
		reflect.DeepEqual(lhs.RoleRef, rhs.RoleRef) {
		return true
	}
	return false
}

// Values implements ResourceInterface
func (s *RoleBindingResource) Values(namespace string) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	roleBindings, err := s.lister.RoleBindings(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, roleBinding := range roleBindings {
		ret[roleBinding.Name] = roleBinding
	}
	return ret, nil
}

// NewRoleBindingDeleteAction creates a generic delete action for the rolebinding.
func NewRoleBindingDeleteAction(
	roleBinding *rbac_v1.RoleBinding, roleBindingResource *RoleBindingResource) *GenericDeleteAction {
	return NewGenericDeleteAction(roleBinding, roleBindingResource)
}

// NewRoleBindingUpsertAction creates a new upsert action for role bindings.
func NewRoleBindingUpsertAction(
	roleBinding *rbac_v1.RoleBinding, roleBindingResource *RoleBindingResource) *GenericUpsertAction {
	return NewGenericUpsertAction(roleBinding, roleBindingResource)
}
