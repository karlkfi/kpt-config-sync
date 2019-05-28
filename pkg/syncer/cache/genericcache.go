// Package cache includes controller caches.
package cache

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// GenericCache extends Cache to better handle fetching objects as Unstructured types.
type GenericCache interface {
	cache.Cache
	// UnstructuredList returns all the resources in the cluster of the given
	// GroupVersionKind for the given namespace. If the namespace is empty, it
	// will return all resources across all namespaces. Namespace should always
	// be set to empty when listing cluster-scoped resources.
	// This method is needed because cache.Cache's List method requires knowing
	// the type of the resource you wanted to list. We always want to return
	// Unstructureds when listing resources on the cluster, whether it's a native
	// or custom resource.
	UnstructuredList(gvk schema.GroupVersionKind, namespace string) ([]*unstructured.Unstructured, error)
}

// GenericResourceCache implements GenericCache.
type GenericResourceCache struct {
	cache.Cache
}

// NewGenericResourceCache returns a new GenericResourceCache.
func NewGenericResourceCache(cache cache.Cache) *GenericResourceCache {
	return &GenericResourceCache{Cache: cache}
}

// UnstructuredList implements GenericCache.
func (c *GenericResourceCache) UnstructuredList(gvk schema.GroupVersionKind,
	namespace string) ([]*unstructured.Unstructured, error) {
	informer, err := c.GetInformerForKind(gvk)
	if err != nil {
		return nil, errors.Wrapf(err, "no informer for %s in the cache", gvk)
	}

	indexer := informer.GetIndexer()
	var objs []interface{}
	if namespace == "" {
		objs = indexer.List()
	} else {
		objs, err = indexer.ByIndex(toolscache.NamespaceIndex, namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "No namespace index for %s in in the cache", gvk)
		}
	}

	var us []*unstructured.Unstructured
	for _, obj := range objs {
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
		u := &unstructured.Unstructured{Object: content}
		u.SetGroupVersionKind(gvk)

		us = append(us, u)
	}
	return us, nil
}
