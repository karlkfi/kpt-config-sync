/*
Copyright 2017 The Stolos Authors.
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
	pn_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	informerspolicynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/policyhierarchy/v1"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
)

// A cache of package quotas that keeps usage and limits in memory for the whole namespace tree.
// The limits and structure are fed from the policyNode informer
// The usage is based on the ResourceQuota informer which has the usage on the leaf nodes
type HierarchicalQuotaCache struct {
	policyNodeInformer    informerspolicynodev1.PolicyNodeInformer
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer

	// Map of namespaces to quota objects
	quotas map[string]*QuotaNode
}

// Contains information about a quota, mainly the resource quota itself, but also its place in the hierarchy.
type QuotaNode struct {
	quota       *core_v1.ResourceQuota // The quota itself, both hard and used.
	parent      string                 // The parent of the namespace for this quota based on the policyNode
	policyspace bool                   // Whether this is a leaf namespace or a non-leaf policynode
}

func NewHierarchicalQuotaCache(policyNodeInformer informerspolicynodev1.PolicyNodeInformer,
	resourceQuotaInformer informerscorev1.ResourceQuotaInformer) (*HierarchicalQuotaCache, error) {
	cache := &HierarchicalQuotaCache{
		policyNodeInformer:    policyNodeInformer,
		resourceQuotaInformer: resourceQuotaInformer,
	}
	err := cache.initCache()

	return cache, err
}

// initCache populates the quotas and parents maps using the current state of the informers.
// TODO(mdruskin): We probably want to add handlers to keep the cache up to date. Right now we need to create a new
//                 cache each time we want to do an admission decision. This might add unnecessary complexity for
//                 not that much performance gain.
func (c *HierarchicalQuotaCache) initCache() error {
	resourceQuotas, err := c.resourceQuotaInformer.Lister().List(labels.SelectorFromSet(StolosQuotaLabels))
	if err != nil {
		return err
	}
	policyNodes, err := c.policyNodeInformer.Lister().List(labels.Everything())
	if err != nil {
		return err
	}
	c.quotas = map[string]*QuotaNode{}

	for _, policyNode := range policyNodes {
		quota := &core_v1.ResourceQuota{
			Status: core_v1.ResourceQuotaStatus{
				Used: core_v1.ResourceList{},
			},
		}
		if policyNode.Spec.Policies.ResourceQuotaV1 != nil {
			quota = &core_v1.ResourceQuota{
				Spec: *policyNode.Spec.Policies.ResourceQuotaV1.Spec.DeepCopy(),
				Status: core_v1.ResourceQuotaStatus{
					Hard: policyNode.Spec.Policies.ResourceQuotaV1.Spec.Hard,
					Used: core_v1.ResourceList{},
				},
			}
		}
		c.quotas[policyNode.Name] = &QuotaNode{
			quota:       quota,
			parent:      policyNode.Spec.Parent,
			policyspace: policyNode.Spec.Policyspace,
		}
	}

	// Set the usage based on the quota informer
	for _, resourceQuota := range resourceQuotas {
		if resourceQuota.Name != ResourceQuotaObjectName {
			continue // Only care about stolos resource quota objects
		}

		quotaNode, exists := c.quotas[resourceQuota.Namespace]
		if !exists {
			glog.Infof("Resource Quota exists for namespace %s not defined in policy nodes", resourceQuota.Namespace)
			continue // This can happen frequently during deletions and while adjusting the tree.
		}
		// For leaf
		resourceQuota.Status.DeepCopyInto(&quotaNode.quota.Status)

		// For all the parents, add up quantities
		parent := quotaNode.parent
		for parent != pn_v1.NoParentNamespace {
			quotaNode, exists := c.quotas[parent]
			if !exists {
				glog.Warningf("Parent namespace %s not defined in policy nodes for child namespace %s",
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
func (c *HierarchicalQuotaCache) Admit(namespace string, newUsageList core_v1.ResourceList) error {
	// Start with the parent of the given namespace
	namespaceQuota, exists := c.quotas[namespace]
	if !exists {
		// No namespace defined in policy nodes so this is not a namespace controlled by stolos.
		return nil
	}
	namespace = namespaceQuota.parent

	// For each level of the hierarchy going up from the direct parent
	for namespace != pn_v1.NoParentNamespace {
		namespaceQuota, exists := c.quotas[namespace]
		if !exists {
			// No namespace defined in policy nodes so this is not a namespace controlled by stolos.
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
					return errors.Errorf("exceeded quota in policyspace %s, requested: %s=%d, limit: %s=%d",
						namespace, resourceName, newTotalUsage.Value(), resourceName, limit.Value())
				}
			}
		}
		namespace = namespaceQuota.parent
	}
	return nil
}

// UpdateLeaf updates the usage quota on a leaf quota namespace and propagates the changes up the tree
// to reflect the new usage in all the parent quotas as well. The function returns a list of namespaces
// that had their quota changed.
func (c *HierarchicalQuotaCache) UpdateLeaf(newQuota core_v1.ResourceQuota) ([]string, error) {
	updatedNamespaces := []string{}
	namespace := newQuota.Namespace
	currentQuota, exists := c.quotas[namespace]
	if !exists {
		return nil, errors.Errorf("Namespace %q does not have a quota in cache", newQuota.Namespace)
	}

	usageDiffs := diffResourceLists(currentQuota.quota.Status.Used, newQuota.Status.Used)

	if len(usageDiffs) == 0 {
		return updatedNamespaces, nil // No diffs, nothing to do.
	}
	// We have diffs, let's update the cache
	for namespace != pn_v1.NoParentNamespace {
		quotaNode, exists := c.quotas[namespace]
		if !exists {
			return nil, errors.Errorf("Parent namespace %q does not have a quota in cache", namespace)
		}

		for resourceName, usageDiff := range usageDiffs {
			if current, exists := quotaNode.quota.Status.Used[resourceName]; exists {
				current.Add(usageDiff)
				quotaNode.quota.Status.Used[resourceName] = current
			} else {
				quotaNode.quota.Status.Used[resourceName] = usageDiff
			}
		}
		if quotaNode.policyspace {
			updatedNamespaces = append(updatedNamespaces, namespace)
		}
		namespace = quotaNode.parent
	}
	return updatedNamespaces, nil
}
