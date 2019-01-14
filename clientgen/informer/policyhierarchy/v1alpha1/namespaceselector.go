/*
Copyright 2019 The Nomos Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	apis "github.com/google/nomos/clientgen/apis"
	internalinterfaces "github.com/google/nomos/clientgen/informer/internalinterfaces"
	v1alpha1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1alpha1"
	policyhierarchy_v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// NamespaceSelectorInformer provides access to a shared informer and lister for
// NamespaceSelectors.
type NamespaceSelectorInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.NamespaceSelectorLister
}

type namespaceSelectorInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewNamespaceSelectorInformer constructs a new informer for NamespaceSelector type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNamespaceSelectorInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNamespaceSelectorInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredNamespaceSelectorInformer constructs a new informer for NamespaceSelector type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredNamespaceSelectorInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NomosV1alpha1().NamespaceSelectors().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NomosV1alpha1().NamespaceSelectors().Watch(options)
			},
		},
		&policyhierarchy_v1alpha1.NamespaceSelector{},
		resyncPeriod,
		indexers,
	)
}

func (f *namespaceSelectorInformer) defaultInformer(client apis.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNamespaceSelectorInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *namespaceSelectorInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&policyhierarchy_v1alpha1.NamespaceSelector{}, f.defaultInformer)
}

func (f *namespaceSelectorInformer) Lister() v1alpha1.NamespaceSelectorLister {
	return v1alpha1.NewNamespaceSelectorLister(f.Informer().GetIndexer())
}
