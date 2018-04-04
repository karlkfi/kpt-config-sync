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

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"

	"github.com/google/nomos/pkg/client/action"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

// NewNamespaceDeleteAction creates a new ReflectiveDeleteAction to delete the given namespace.
func NewNamespaceDeleteAction(
	namespace string,
	client kubernetes.Interface,
	lister listers_core_v1.NamespaceLister) *action.ReflectiveDeleteAction {
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(core_v1.Namespace{}),
		KindPlural: action.Plural(core_v1.Namespace{}),
		Group:      core_v1.SchemeGroupVersion.Group,
		Version:    core_v1.SchemeGroupVersion.Version,
		EqualSpec:  NamespacesEqual,
		Client:     client.CoreV1(),
		Lister:     lister,
	}
	return action.NewReflectiveDeleteAction("", namespace, spec)
}

// NewNamespaceUpsertAction creates a new ReflectiveUpsertAction for the given namespace.
func NewNamespaceUpsertAction(
	namespace string,
	uid types.UID,
	labels map[string]string,
	client kubernetes.Interface,
	lister listers_core_v1.NamespaceLister) *action.ReflectiveUpsertAction {
	blockOwnerDeletion := true
	controller := true
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(core_v1.Namespace{}),
		KindPlural: action.Plural(core_v1.Namespace{}),
		Group:      core_v1.SchemeGroupVersion.Group,
		Version:    core_v1.SchemeGroupVersion.Version,
		EqualSpec:  NamespacesEqual,
		Client:     client.CoreV1(),
		Lister:     lister,
	}
	ns := &core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   namespace,
			Labels: labels,
			OwnerReferences: []meta_v1.OwnerReference{
				{
					APIVersion:         policyhierarchy_v1.SchemeGroupVersion.String(),
					Kind:               "PolicyNode",
					Name:               namespace,
					UID:                uid,
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &controller,
				},
			},
		},
	}
	return action.NewReflectiveUpsertAction("", namespace, ns, spec)
}

// NamespacesEqual returns true if the two Namespaces have the same owner references.
func NamespacesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lNamespace := lhs.(*core_v1.Namespace)
	rNamespace := rhs.(*core_v1.Namespace)
	return reflect.DeepEqual(lNamespace.OwnerReferences, rNamespace.OwnerReferences)
}
