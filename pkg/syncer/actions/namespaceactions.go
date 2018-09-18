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
// Reviewed by sunilarora

package actions

import (
	"reflect"

	"github.com/google/nomos/pkg/client/action"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

func nsSpec(
	client kubernetes.Interface,
	lister listerscorev1.NamespaceLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(corev1.Namespace),
		corev1.SchemeGroupVersion,
		NamespacesEqual,
		client.CoreV1(),
		lister)
}

// NewNamespaceDeleteAction creates a new ReflectiveDeleteAction to delete the given namespace.
func NewNamespaceDeleteAction(
	namespace string,
	client kubernetes.Interface,
	lister listerscorev1.NamespaceLister) *action.ReflectiveDeleteAction {
	return action.NewReflectiveDeleteAction(
		"", namespace, nsSpec(client, lister))
}

// NewNamespaceUpsertAction creates a new ReflectiveUpsertAction for the given namespace.
func NewNamespaceUpsertAction(
	namespace *corev1.Namespace,
	client kubernetes.Interface,
	lister listerscorev1.NamespaceLister) *action.ReflectiveUpsertAction {
	return action.NewReflectiveUpsertAction(
		"", namespace.Name, namespace, nsSpec(client, lister))
}

// UpdateObjectFunction is passed to NewNamespaceUpdateAction for updating a namespace
type UpdateObjectFunction func(old runtime.Object) (runtime.Object, error)

// SetNamespaceLabelsFunc produces a function that sets labels to specific values while preserving
// the rest of the labels on the namespace
func SetNamespaceLabelsFunc(labels map[string]string) UpdateObjectFunction {
	return func(old runtime.Object) (runtime.Object, error) {
		oldNs := old.(*corev1.Namespace)
		newNs := oldNs.DeepCopy()
		dirty := false
		for key, value := range labels {
			if oldValue, found := newNs.Labels[key]; !found || oldValue != value {
				dirty = true
				newNs.Labels[key] = value
			}
		}
		if !dirty {
			return nil, action.NoUpdateNeeded()
		}
		return newNs, nil
	}
}

// NewNamespaceUpdateAction creates a new ReflectiveUpdateAction for the namespace.
func NewNamespaceUpdateAction(
	namespace string,
	updateFunction func(old runtime.Object) (runtime.Object, error),
	client kubernetes.Interface,
	lister listerscorev1.NamespaceLister) *action.ReflectiveUpdateAction {
	return action.NewReflectiveUpdateAction("", namespace, updateFunction, nsSpec(client, lister))
}

// NewNamespaceCreateAction creates a new ReflectiveCreateAction for the namespace.
func NewNamespaceCreateAction(
	namespace *corev1.Namespace,
	client kubernetes.Interface,
	lister listerscorev1.NamespaceLister) *action.ReflectiveCreateAction {
	return action.NewReflectiveCreateAction(
		"", namespace.Name, namespace, nsSpec(client, lister))
}

// NamespacesEqual returns true if the two Namespaces have the same owner references.
func NamespacesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lNamespace := lhs.(*corev1.Namespace)
	rNamespace := rhs.(*corev1.Namespace)
	return reflect.DeepEqual(lNamespace.OwnerReferences, rNamespace.OwnerReferences)
}
