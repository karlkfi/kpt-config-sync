/*
Copyright 2019 The CSP Config Management Authors.

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

package v1

import (
	time "time"

	apis "github.com/google/nomos/clientgen/apis"
	internalinterfaces "github.com/google/nomos/clientgen/informer/internalinterfaces"
	v1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ClusterSelectorInformer provides access to a shared informer and lister for
// ClusterSelectors.
type ClusterSelectorInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ClusterSelectorLister
}

type clusterSelectorInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterSelectorInformer constructs a new informer for ClusterSelector type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterSelectorInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterSelectorInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterSelectorInformer constructs a new informer for ClusterSelector type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterSelectorInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().ClusterSelectors().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().ClusterSelectors().Watch(options)
			},
		},
		&configmanagementv1.ClusterSelector{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterSelectorInformer) defaultInformer(client apis.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterSelectorInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterSelectorInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&configmanagementv1.ClusterSelector{}, f.defaultInformer)
}

func (f *clusterSelectorInformer) Lister() v1.ClusterSelectorLister {
	return v1.NewClusterSelectorLister(f.Informer().GetIndexer())
}
