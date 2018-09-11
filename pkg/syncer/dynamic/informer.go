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

// Package dynamic includes listers and informers for dynamically specified resources.
package dynamic

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

// InformerProvider provides access to a shared informer and lister for
// an arbitrary resource.
// TODO(116230169): Consider making this an informer factory if we want to have
// multiple informers sharing the same cache.
type InformerProvider struct {
	informer cache.SharedIndexInformer
	lister   Lister
}

// NewInformerProvider returns an InformerProvider for the resource, using a
// dynamic client. For cluster-scoped resources, namespace is ignored. When a
// namespace is not specified for namespace-scoped resources, the informer
// operates across all namespaces.
func NewInformerProvider(client *dynamic.Client, resource *metav1.APIResource, namespace string, resyncPeriod time.Duration,
	indexers cache.Indexers) InformerProvider {
	resourceClient := client.Resource(resource, namespace)
	gr := schema.GroupResource{
		Group:    resource.Group,
		Resource: resource.SingularName,
	}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc:  resourceClient.List,
			WatchFunc: resourceClient.Watch,
		},
		&unstructured.Unstructured{},
		resyncPeriod,
		indexers,
	)
	return InformerProvider{informer, newLister(informer.GetIndexer(), gr)}
}

// NewDefaultInformerProvider returns an InformerProvider with a default indexer.
func NewDefaultInformerProvider(client *dynamic.Client, resource *metav1.APIResource, namespace string,
	resyncPeriod time.Duration) InformerProvider {
	return NewInformerProvider(client, resource, namespace, resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

// Informer returns the informer provided by the InformerProvider
func (f *InformerProvider) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the lister provided by the InformerProvider
func (f *InformerProvider) Lister() Lister {
	return f.lister
}
