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

package v1

import (
	internalinterfaces "github.com/google/nomos/clientgen/informer/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ClusterConfigs returns a ClusterConfigInformer.
	ClusterConfigs() ClusterConfigInformer
	// ClusterSelectors returns a ClusterSelectorInformer.
	ClusterSelectors() ClusterSelectorInformer
	// HierarchicalQuotas returns a HierarchicalQuotaInformer.
	HierarchicalQuotas() HierarchicalQuotaInformer
	// HierarchyConfigs returns a HierarchyConfigInformer.
	HierarchyConfigs() HierarchyConfigInformer
	// NamespaceConfigs returns a NamespaceConfigInformer.
	NamespaceConfigs() NamespaceConfigInformer
	// NamespaceSelectors returns a NamespaceSelectorInformer.
	NamespaceSelectors() NamespaceSelectorInformer
	// Repos returns a RepoInformer.
	Repos() RepoInformer
	// Syncs returns a SyncInformer.
	Syncs() SyncInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// ClusterConfigs returns a ClusterConfigInformer.
func (v *version) ClusterConfigs() ClusterConfigInformer {
	return &clusterConfigInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterSelectors returns a ClusterSelectorInformer.
func (v *version) ClusterSelectors() ClusterSelectorInformer {
	return &clusterSelectorInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// HierarchicalQuotas returns a HierarchicalQuotaInformer.
func (v *version) HierarchicalQuotas() HierarchicalQuotaInformer {
	return &hierarchicalQuotaInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// HierarchyConfigs returns a HierarchyConfigInformer.
func (v *version) HierarchyConfigs() HierarchyConfigInformer {
	return &hierarchyConfigInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// NamespaceConfigs returns a NamespaceConfigInformer.
func (v *version) NamespaceConfigs() NamespaceConfigInformer {
	return &namespaceConfigInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// NamespaceSelectors returns a NamespaceSelectorInformer.
func (v *version) NamespaceSelectors() NamespaceSelectorInformer {
	return &namespaceSelectorInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Repos returns a RepoInformer.
func (v *version) Repos() RepoInformer {
	return &repoInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Syncs returns a SyncInformer.
func (v *version) Syncs() SyncInformer {
	return &syncInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
