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

// RoleResource is an implementation of ResourceInterface for roles.
type RoleResource struct {
	client kubernetes.Interface
	lister listers_rbac_v1.RoleLister
}

// RoleResource implements ResourceInterface
var _ ResourceInterface = &RoleResource{}

// NewRoleResource creates a role binding resource from the kubernetes client objects.
func NewRoleResource(
	client kubernetes.Interface,
	lister listers_rbac_v1.RoleLister) *RoleResource {
	return &RoleResource{
		client: client,
		lister: lister,
	}
}

// Create implements ResourceInterface
func (s *RoleResource) Create(obj interface{}) (interface{}, error) {
	role := obj.(*rbac_v1.Role)
	return s.client.RbacV1().Roles(role.Namespace).Create(role)
}

// Delete implements ResourceInterface
func (s *RoleResource) Delete(obj interface{}) error {
	role := obj.(*rbac_v1.Role)
	return s.client.RbacV1().Roles(role.Namespace).Delete(role.Name, &meta_v1.DeleteOptions{})
}

// Get implements ResourceInterface
func (s *RoleResource) Get(obj interface{}) (interface{}, error) {
	role := obj.(*rbac_v1.Role)
	return s.client.RbacV1().Roles(role.Namespace).Get(role.Name, meta_v1.GetOptions{})
}

// Update implements ResourceInterface
func (s *RoleResource) Update(oldObj interface{}, newObj interface{}) (interface{}, error) {
	nrb := newObj.(*rbac_v1.Role)
	nrb.ResourceVersion = oldObj.(*rbac_v1.Role).ResourceVersion
	return s.client.RbacV1().Roles(nrb.Namespace).Update(nrb)
}

// Type implements ResourceInterface
func (s *RoleResource) Type() string {
	return "role"
}

// Kind implements ResourceInterface
func (s *RoleResource) Kind() string {
	return "Role"
}

// Group implements ResourceInterface
func (s *RoleResource) Group(obj interface{}) string {
	return rbac_v1.SchemeGroupVersion.Group
}

// Version implements ResourceInterface
func (s RoleResource) Version(obj interface{}) string {
	return rbac_v1.SchemeGroupVersion.Version
}

// Name implements ResourceInterface
func (s *RoleResource) Name(obj interface{}) string {
	return obj.(*rbac_v1.Role).Name
}

// Namespace implements ResourceInterface
func (s *RoleResource) Namespace(obj interface{}) string {
	return obj.(*rbac_v1.Role).Namespace
}

// Equal implements ResourceInterface
func (s *RoleResource) Equal(lhsObj interface{}, rhsObj interface{}) bool {
	lhs := lhsObj.(*rbac_v1.Role)
	rhs := rhsObj.(*rbac_v1.Role)
	if MetaEquals(lhs.ObjectMeta, rhs.ObjectMeta) &&
		reflect.DeepEqual(lhs.Rules, rhs.Rules) {
		return true
	}
	return false
}

// Values implements ResourceInterface
func (s *RoleResource) Values(namespace string) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	roles, err := s.lister.Roles(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		ret[role.Name] = role
	}
	return ret, nil
}

// NewRoleDeleteAction creates a generic delete action for the rolebinding.
func NewRoleDeleteAction(
	role *rbac_v1.Role, roleResource *RoleResource) *GenericDeleteAction {
	return NewGenericDeleteAction(role, roleResource)
}

// NewRoleUpsertAction creates a new upsert action for role bindings.
func NewRoleUpsertAction(
	role *rbac_v1.Role, roleResource *RoleResource) *GenericUpsertAction {
	return NewGenericUpsertAction(role, roleResource)
}
