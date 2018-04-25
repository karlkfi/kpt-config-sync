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

	"github.com/google/nomos/pkg/client/action"
	rbac_v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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

// NewRoleDeleteAction returns a new delete action for the rolebinding.
func NewRoleDeleteAction(
	namespace, name string, roleResource *RoleResource) *action.ReflectiveDeleteAction {
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(rbac_v1.Role{}),
		KindPlural: action.Plural(rbac_v1.Role{}),
		Group:      rbac_v1.SchemeGroupVersion.Group,
		Version:    rbac_v1.SchemeGroupVersion.Version,
		EqualSpec:  RolesEqual,
		Client:     roleResource.client.RbacV1(),
		Lister:     roleResource.lister,
	}
	return action.NewReflectiveDeleteAction(namespace, name, spec)
}

// NewRoleUpsertAction returns a new upsert action for role bindings.
func NewRoleUpsertAction(
	role *rbac_v1.Role, roleResource *RoleResource) *action.ReflectiveUpsertAction {
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(rbac_v1.Role{}),
		KindPlural: action.Plural(rbac_v1.Role{}),
		Group:      rbac_v1.SchemeGroupVersion.Group,
		Version:    rbac_v1.SchemeGroupVersion.Version,
		EqualSpec:  RolesEqual,
		Client:     roleResource.client.RbacV1(),
		Lister:     roleResource.lister,
	}
	return action.NewReflectiveUpsertAction(role.Namespace, role.Name, role, spec)
}

// RolesEqual returns true if the rules in two Role objects are equal.
func RolesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lRole := lhs.(*rbac_v1.Role)
	rRole := rhs.(*rbac_v1.Role)
	return reflect.DeepEqual(lRole.Rules, rRole.Rules)
}
