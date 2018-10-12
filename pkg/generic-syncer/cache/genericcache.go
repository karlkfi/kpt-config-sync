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

// Package cache includes controller caches.
package cache

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// GenericCache extends Cache to better handle fetching objects as Unstructured types.
type GenericCache struct {
	cache.Cache
}

// NewGenericCache returns a new GenericCache.
func NewGenericCache(cache cache.Cache) *GenericCache {
	return &GenericCache{Cache: cache}
}

// List returns all the resources in the cluster of the given GroupVersionKind.
// This is needed because cache.Cache's List method requires knowing the type
// of the resource you wanted to list. We always want to return Unstructureds
// when listing resources on the cluster, whether it's a native or custom
// resource.
func (c *GenericCache) List(gvk schema.GroupVersionKind) ([]*unstructured.Unstructured, error) {
	informer, err := c.GetInformerForKind(gvk)
	if err != nil {
		return nil, errors.Wrapf(err, "no informer for %s in the cache", gvk)
	}

	var us []*unstructured.Unstructured
	for _, obj := range informer.GetIndexer().List() {
		if u, ok := obj.(*unstructured.Unstructured); ok {
			// Object is registered as Unstructured; no conversion is needed.
			us = append(us, u)
			continue
		}

		content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			// All resources the Syncer syncs need to be convertible to Unstructured.
			panic(errors.Wrapf(err, "cannot convert %s to Unstructured object", gvk))
		}
		us = append(us, &unstructured.Unstructured{Object: content})
	}
	return us, nil
}
