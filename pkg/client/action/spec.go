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

package action

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ReflectiveActionSpec represents information and objects needed for performing actions on a given
// GroupVersionKind.
type ReflectiveActionSpec struct {
	// The resource type name.
	Resource string
	// The plural of a kind, eg, Roles, RoleBindings, Policies as used for getting the client from
	// the generated code.
	KindPlural string
	// The group name of the GroupVersionKind being acted on.
	Group string
	// The version of the GroupVersionKind being acted on.
	Version string
	// EqualSpec is the per-Kind equal operation that check for equality for the spec of an object.
	// Meta fields (ObjectMeta and TypeMeta) equality is done automatically and need not be done
	// by this function.
	EqualSpec func(lhs runtime.Object, rhs runtime.Object) bool
	// Client is the client-gen generated stub for the given API group, for example:
	// kubernetesClient.RbacV1() or kubernetesClient.CoreV1()
	Client interface{}
	// Lister is the lister from the generated informer for the given, example:
	// kubernetesInformerFactory.Rbac().V1().ClusterRoles().Lister()
	Lister interface{}
}

// NewSpec creates a new spec from an instance of the api type, the groupversion, an equals function,
// the client and the lister.
func NewSpec(
	instance runtime.Object,
	groupVersion schema.GroupVersion,
	equals func(lhs runtime.Object, rhs runtime.Object) bool,
	client interface{},
	lister interface{}) *ReflectiveActionSpec {
	return &ReflectiveActionSpec{
		Resource:   LowerPlural(instance),
		KindPlural: Plural(instance),
		Group:      groupVersion.Group,
		Version:    groupVersion.Version,
		EqualSpec:  equals,
		Client:     client,
		Lister:     lister,
	}
}

// Equal returns true if the two objects have equivalent per-kind spec equality and the
// labels and annotations are a superset of the declared labels and annotations.
func (s *ReflectiveActionSpec) Equal(declared runtime.Object, actual runtime.Object) bool {
	if !s.EqualSpec(actual, declared) {
		return false
	}
	return ObjectMetaSubset(actual, declared)
}

// List will list the given namespace. Cluster level resources should pass empty string as the
// namespace.
// Example of what this is roughly doing:
// -- For cluster scoped resources --
// return kubernetesInformerFactory.Rbac().V1().ClusterRoles().Lister().List(selector)
// -- For namesapce scoped resources --
// return kubernetesInformerFactory.Rbac().V1().Roles().Lister().Roles(namespace).List(selector)
func (s *ReflectiveActionSpec) List(namespace string, selector labels.Selector) ([]runtime.Object, error) {
	lister := s.listerValue(namespace)
	listMethod := lister.MethodByName("List")
	listArgs := []reflect.Value{reflect.ValueOf(selector)}
	returnValues := listMethod.Call(listArgs)
	if len(returnValues) != 2 {
		panic(fmt.Sprintf("list call returned invalid number of args %v", returnValues))
	}

	if !returnValues[1].IsNil() {
		return nil, returnValues[1].Interface().(error)
	}

	objValues := returnValues[0]
	objs := make([]runtime.Object, objValues.Len())
	for i := 0; i < objValues.Len(); i++ {
		objs[i] = objValues.Index(i).Interface().(runtime.Object)
	}
	return objs, nil
}

// listerValue returns the appropriate lister taking into account cluster / namespace scoping
// Example of what this is effectively doing:
// -- For cluster scoped resources --
// s.Lister := kubernetesInformerFactory.Rbac().V1().ClusterRoles().Lister()
// return s.spec.Lister
// -- For namesapce scoped resources --
// s.Lister := kubernetesInformerFactory.Rbac().V1().Roles().Lister()
// return s.spec.Lister.Roles(s.namespace)
func (s *ReflectiveActionSpec) listerValue(namespace string) reflect.Value {
	listerValue := reflect.ValueOf(s.Lister)
	if namespace == "" {
		return listerValue
	}

	methodValue := listerValue.MethodByName(s.KindPlural)
	listerReturnValues := methodValue.Call([]reflect.Value{reflect.ValueOf(namespace)})
	if len(listerReturnValues) != 1 {
		panic(fmt.Sprintf("Getting lister returned invalid number of values"))
	}
	return listerReturnValues[0]
}
