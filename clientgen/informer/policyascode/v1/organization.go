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

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	time "time"

	apis "github.com/google/nomos/clientgen/apis"
	internalinterfaces "github.com/google/nomos/clientgen/informer/internalinterfaces"
	v1 "github.com/google/nomos/clientgen/listers/policyascode/v1"
	policyascode_v1 "github.com/google/nomos/pkg/api/policyascode/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// OrganizationInformer provides access to a shared informer and lister for
// Organizations.
type OrganizationInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.OrganizationLister
}

type organizationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewOrganizationInformer constructs a new informer for Organization type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewOrganizationInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredOrganizationInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredOrganizationInformer constructs a new informer for Organization type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredOrganizationInformer(client apis.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BespinV1().Organizations().List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BespinV1().Organizations().Watch(options)
			},
		},
		&policyascode_v1.Organization{},
		resyncPeriod,
		indexers,
	)
}

func (f *organizationInformer) defaultInformer(client apis.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredOrganizationInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *organizationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&policyascode_v1.Organization{}, f.defaultInformer)
}

func (f *organizationInformer) Lister() v1.OrganizationLister {
	return v1.NewOrganizationLister(f.Informer().GetIndexer())
}
