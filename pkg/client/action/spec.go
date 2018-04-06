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
	"reflect"

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
		Resource:   LowerPlural(reflect.TypeOf(instance)),
		KindPlural: Plural(reflect.TypeOf(instance)),
		Group:      groupVersion.Group,
		Version:    groupVersion.Version,
		EqualSpec:  equals,
		Client:     client,
		Lister:     lister,
	}
}

// Equal returns true if the two objects have equivalent per-kind spec equality and the
// labels and annotations are a superset of the declared labels and annotations.
func (s ReflectiveActionSpec) Equal(declared runtime.Object, actual runtime.Object) bool {
	if !s.EqualSpec(actual, declared) {
		return false
	}
	return ObjectMetaSubset(actual, declared)
}
