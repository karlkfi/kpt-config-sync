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

// SyncInformer provides access to a shared informer and lister for
// Syncs.
type SyncInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.SyncLister
}

type syncInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewSyncInformer constructs a new informer for Sync type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSyncInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSyncInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredSyncInformer constructs a new informer for Sync type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSyncInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().Syncs().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().Syncs().Watch(options)
			},
		},
		&configmanagementv1.Sync{},
		resyncPeriod,
		indexers,
	)
}

func (f *syncInformer) defaultInformer(client apis.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSyncInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *syncInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&configmanagementv1.Sync{}, f.defaultInformer)
}

func (f *syncInformer) Lister() v1.SyncLister {
	return v1.NewSyncLister(f.Informer().GetIndexer())
}
