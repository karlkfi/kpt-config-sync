// Package cache includes controller caches.
package cache

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	UnstructuredList(ctx context.Context, gvk schema.GroupVersionKind,
		namespace string) ([]*unstructured.Unstructured, error)
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
func (c *GenericResourceCache) UnstructuredList(ctx context.Context, gvk schema.GroupVersionKind,
	namespace string) ([]*unstructured.Unstructured, error) {
	if !strings.HasSuffix(gvk.Kind, "List") {
		gvk.Kind = gvk.Kind + "List"
	}

	ul := &unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(gvk)
	if namespace == "" {
		err := c.List(ctx, ul)
		if err != nil {
			return nil, errors.Wrapf(err, "Error while fetching Unstructured List for GVK %v", gvk)
		}
	} else {
		err := c.List(ctx, ul, client.InNamespace(namespace))
		if err != nil {
			return nil, errors.Wrapf(err, "No namespace index for %s in in the cache", gvk)
		}
	}

	// The existing API uses arrays of pointers to Unstructureds; Items is actual structs
	// we oblige and convert here for the return array.
	var uls []*unstructured.Unstructured
	for i := 0; i < len(ul.Items); i++ {
		uls = append(uls, &ul.Items[i])
	}

	return uls, nil
}
