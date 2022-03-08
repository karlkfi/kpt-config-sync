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

// ClusterConfigInformer provides access to a shared informer and lister for
// ClusterConfigs.
type ClusterConfigInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ClusterConfigLister
}

type clusterConfigInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterConfigInformer constructs a new informer for ClusterConfig type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterConfigInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterConfigInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterConfigInformer constructs a new informer for ClusterConfig type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterConfigInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().ClusterConfigs().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ConfigmanagementV1().ClusterConfigs().Watch(context.TODO(), options)
			},
		},
		&configmanagementv1.ClusterConfig{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterConfigInformer) defaultInformer(client apis.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterConfigInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterConfigInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&configmanagementv1.ClusterConfig{}, f.defaultInformer)
}

func (f *clusterConfigInformer) Lister() v1.ClusterConfigLister {
	return v1.NewClusterConfigLister(f.Informer().GetIndexer())
}
