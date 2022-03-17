// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	apis "kpt.dev/configsync/clientgen/apis"
	internalinterfaces "kpt.dev/configsync/clientgen/informer/internalinterfaces"
	v1 "kpt.dev/configsync/clientgen/listers/configmanagement/v1"
	configmanagementv1 "kpt.dev/configsync/pkg/api/configmanagement/v1"
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
				return client.ConfigmanagementV1().ClusterSelectors().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().ClusterSelectors().Watch(context.TODO(), options)
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
