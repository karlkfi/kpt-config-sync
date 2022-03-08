// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	apis "kpt.dev/configsync/clientgen/apis"
	internalinterfaces "kpt.dev/configsync/clientgen/informer/internalinterfaces"
	v1 "kpt.dev/configsync/clientgen/listers/configmanagement/v1"
	configmanagementv1 "kpt.dev/configsync/pkg/api/configmanagement/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// HierarchyConfigInformer provides access to a shared informer and lister for
// HierarchyConfigs.
type HierarchyConfigInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.HierarchyConfigLister
}

type hierarchyConfigInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewHierarchyConfigInformer constructs a new informer for HierarchyConfig type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewHierarchyConfigInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredHierarchyConfigInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredHierarchyConfigInformer constructs a new informer for HierarchyConfig type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredHierarchyConfigInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().HierarchyConfigs().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().HierarchyConfigs().Watch(context.TODO(), options)
			},
		},
		&configmanagementv1.HierarchyConfig{},
		resyncPeriod,
		indexers,
	)
}

func (f *hierarchyConfigInformer) defaultInformer(client apis.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredHierarchyConfigInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *hierarchyConfigInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&configmanagementv1.HierarchyConfig{}, f.defaultInformer)
}

func (f *hierarchyConfigInformer) Lister() v1.HierarchyConfigLister {
	return v1.NewHierarchyConfigLister(f.Informer().GetIndexer())
}
