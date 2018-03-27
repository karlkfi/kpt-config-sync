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

// NewRoleBindingDeleteAction returns a reflective delete action for the rolebinding.
func NewRoleBindingDeleteAction(
	roleBinding *rbac_v1.RoleBinding, roleBindingResource *RoleBindingResource) *action.ReflectiveDeleteAction {
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(rbac_v1.RoleBinding{}),
		KindPlural: action.Plural(rbac_v1.RoleBinding{}),
		Group:      rbac_v1.SchemeGroupVersion.Group,
		Version:    rbac_v1.SchemeGroupVersion.Version,
		EqualSpec:  RoleBindingsEqual,
		Client:     roleBindingResource.client.RbacV1(),
		Lister:     roleBindingResource.lister,
	}
	return action.NewReflectiveDeleteAction(roleBinding.Namespace, roleBinding.Name, spec)
}

// NewRoleBindingUpsertAction returns a new reflective upsert action for role bindings.
func NewRoleBindingUpsertAction(
	roleBinding *rbac_v1.RoleBinding, roleBindingResource *RoleBindingResource) *action.ReflectiveUpsertAction {
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(rbac_v1.RoleBinding{}),
		KindPlural: action.Plural(rbac_v1.RoleBinding{}),
		Group:      rbac_v1.SchemeGroupVersion.Group,
		Version:    rbac_v1.SchemeGroupVersion.Version,
		EqualSpec:  RoleBindingsEqual,
		Client:     roleBindingResource.client.RbacV1(),
		Lister:     roleBindingResource.lister,
	}
	return action.NewReflectiveUpsertAction(roleBinding.Namespace, roleBinding.Name, roleBinding, spec)
}

func RoleBindingsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lRoleBinding := lhs.(*rbac_v1.RoleBinding)
	rRoleBinding := rhs.(*rbac_v1.RoleBinding)
	return reflect.DeepEqual(lRoleBinding.Subjects, rRoleBinding.Subjects) &&
		reflect.DeepEqual(lRoleBinding.RoleRef, rRoleBinding.RoleRef)
}
