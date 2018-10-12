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

// Package object exists to make converting between the meta v1 Object interface and the
// runtime.Object interface less painful.  Both generally refer to API types, but for some reason
// they are separate interfaces and various libraries like using one or the other, so this exists
// to help with converting between the two without duplicating the same for loop a bunch of times.
package object

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RuntimeToMeta converts a list of runtime.Object to a list of metav1.Object
func RuntimeToMeta(runtimeObjs []runtime.Object) []metav1.Object {
	metaObjs := make([]metav1.Object, len(runtimeObjs))
	for idx := range runtimeObjs {
		metaObjs[idx] = runtimeObjs[idx].(metav1.Object)
	}
	return metaObjs
}

// UnstructuredToMeta converts a list of runtime.Object to a list of metav1.Object
func UnstructuredToMeta(unsObjs []*unstructured.Unstructured) []metav1.Object {
	metaObjs := make([]metav1.Object, len(unsObjs))
	for idx := range unsObjs {
		metaObjs[idx] = metav1.Object(unsObjs[idx])
	}
	return metaObjs
}

// RuntimeToMetaMap converts a map of string to runtime.Object to a map of string to metav1.Object
func RuntimeToMetaMap(runMap map[string]runtime.Object) map[string]metav1.Object {
	metaMap := map[string]metav1.Object{}
	for k, v := range runMap {
		metaMap[k] = v.(metav1.Object)
	}
	return metaMap
}

// MetaToRuntime converts a list of metav1.Object to a list of runtime.Object
func MetaToRuntime(metaObjs []metav1.Object) []runtime.Object {
	runtimeObjs := make([]runtime.Object, len(metaObjs))
	for idx := range runtimeObjs {
		runtimeObjs[idx] = metaObjs[idx].(runtime.Object)
	}
	return runtimeObjs
}

// MetaToRuntimeMap converts a map of string to metav1.Object to a map of string to runtime.Object
func MetaToRuntimeMap(metaMap map[string]metav1.Object) map[string]runtime.Object {
	runMap := map[string]runtime.Object{}
	for k, v := range metaMap {
		runMap[k] = v.(runtime.Object)
	}
	return runMap
}
