/*
Copyright 2017 The CSP Config Management Authors.
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

package resourcequota

import (
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	informersv1 "github.com/google/nomos/clientgen/informer/configmanagement/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
)

// HierarchicalQuotaCache is A cache of package quotas that keeps usage and limits in memory for
// the whole namespace tree. The limits and structure are fed from the namespaceConfig informer
// The usage is based on the ResourceQuota informer which has the usage on the leaf nodes
type HierarchicalQuotaCache struct {
	resourceQuotaInformer     informerscorev1.ResourceQuotaInformer
	hierarchicalQuotaInformer informersv1.HierarchicalQuotaInformer

	// Map of namespaces to quota objects
	quotas map[string]*QuotaNode
}

// QuotaNode contains information about a quota, mainly the resource quota itself, but also its place in the hierarchy.
type QuotaNode struct {
	quota  *corev1.ResourceQuota // The quota itself, both hard and used.
	parent string                // The parent of the namespace for this quota based on the namespaceConfig
}

// NewHierarchicalQuotaCache returns the hierarchical quota cache
func NewHierarchicalQuotaCache(
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer,
	hierarchicalQuotaInformer informersv1.HierarchicalQuotaInformer) (*HierarchicalQuotaCache, error) {
	cache := &HierarchicalQuotaCache{
		resourceQuotaInformer:     resourceQuotaInformer,
		hierarchicalQuotaInformer: hierarchicalQuotaInformer,
	}
	err := cache.initCache()

	return cache, err
}

// initQuotaLimits populates the quota limits set by the HierarchicalQuota object from the repo.
func (c *HierarchicalQuotaCache) initQuotaLimits(node *v1.HierarchicalQuotaNode, parent string) {
	quota := &corev1.ResourceQuota{
		Status: corev1.ResourceQuotaStatus{
			Used: corev1.ResourceList{},
		},
	}

	if node.ResourceQuotaV1 != nil {
		quota = &corev1.ResourceQuota{
			Spec: *node.ResourceQuotaV1.Spec.DeepCopy(),
			Status: corev1.ResourceQuotaStatus{
				Hard: node.ResourceQuotaV1.Spec.Hard,
				Used: corev1.ResourceList{},
			},
		}
	}

	c.quotas[node.Name] = &QuotaNode{
		quota:  quota,
		parent: parent,
	}

	if node.Type == v1.HierarchyNodeAbstractNamespace {
		for _, child := range node.Children {
			c.initQuotaLimits(&child, node.Name)
		}
	}
}

// initCache populates the quotas and parents maps using the current state of the informers.
// TODO(mdruskin): We probably want to add handlers to keep the cache up to date. Right now we need to create a new
//                 cache each time we want to do an admission decision. This might add unnecessary complexity for
//                 not that much performance gain.
func (c *HierarchicalQuotaCache) initCache() error {
	resourceQuotas, err := c.resourceQuotaInformer.Lister().List(labels.SelectorFromSet(ConfigManagementQuotaLabels))
	if err != nil {
		return err
	}
	c.quotas = map[string]*QuotaNode{}

	hierarchicalQuota, err := c.hierarchicalQuotaInformer.Lister().Get(ResourceQuotaHierarchyName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			glog.Warningf("Hierarchical Quota object %s does not exist", ResourceQuotaObjectName)
			return nil
		}
		return err
	}
	// Set the quota limits from the repo definition
	c.initQuotaLimits(&hierarchicalQuota.Spec.Hierarchy, v1.NoParentNamespace)

	// Set the usage based on the quota informer
	for _, resourceQuota := range resourceQuotas {
		if resourceQuota.Name != ResourceQuotaObjectName {
			continue // Only care about nomos resource quota objects
		}

		quotaNode, exists := c.quotas[resourceQuota.Namespace]
		if !exists {
			glog.Infof("Resource Quota exists for namespace %q which is not defined in a NamespaceConfig", resourceQuota.Namespace)
			continue // This can happen frequently during deletions and while adjusting the tree.
		}
		// For leaf
		resourceQuota.Status.DeepCopyInto(&quotaNode.quota.Status)

		// For all the parents, add up quantities
		parent := quotaNode.parent
		for parent != v1.NoParentNamespace {
			quotaNode, exists := c.quotas[parent]
			if !exists {
				glog.Warningf("Parent namespace %s not defined in NamespaceConfig for child namespace %s",
					parent, resourceQuota.Namespace)
				break
			}
			for resourceName, quantity := range resourceQuota.Status.Used {
				if current, exists := quotaNode.quota.Status.Used[resourceName]; exists {
					current.Add(quantity)
					quotaNode.quota.Status.Used[resourceName] = current
				} else {
					quotaNode.quota.Status.Used[resourceName] = quantity
				}
			}
			parent = quotaNode.parent
		}
	}
	return nil
}

// Admit checks whether the new usage can be applied to the provided namespace's ancestors.
// If cannot admit returns an error describing the quota that was violated.
func (c *HierarchicalQuotaCache) Admit(namespace string, newUsageList corev1.ResourceList) error {
	// Start with the parent of the given namespace
	namespaceQuota, exists := c.quotas[namespace]
	if !exists {
		// No namespace defined in NamespaceConfigs so this is not a namespace controlled by nomos.
		return nil
	}
	namespace = namespaceQuota.parent

	// For each level of the hierarchy going up from the direct parent
	for namespace != v1.NoParentNamespace {
		namespaceQuota, exists := c.quotas[namespace]
		if !exists {
			// No namespace defined in NamespaceConfigs so this is not a namespace controlled by nomos.
			return nil
		}
		for resourceName, newUsage := range newUsageList {
			current, exists := namespaceQuota.quota.Status.Used[resourceName]
			if !exists {
				current = resource.MustParse("0")
			}
			limit, exists := namespaceQuota.quota.Spec.Hard[resourceName]
			if exists {
				newTotalUsage := current.Copy()
				newTotalUsage.Add(newUsage)
				if newTotalUsage.Cmp(limit) > 0 {
					Metrics.Violations.WithLabelValues(resourceName.String()).Inc()
					return errors.Errorf("exceeded quota in %s, requested: %s=%d, limit: %s=%d",
						namespace, resourceName, newTotalUsage.Value(), resourceName, limit.Value())
				}

				Metrics.Usage.WithLabelValues(resourceName.String()).Set(float64(newTotalUsage.Value()))
			}
		}
		namespace = namespaceQuota.parent
	}
	return nil
}
