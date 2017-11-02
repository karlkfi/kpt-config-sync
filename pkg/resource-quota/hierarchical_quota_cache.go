/*
Copyright 2017 The Kubernetes Authors.
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

package resource_quota

import (
	"github.com/golang/glog"
	pn_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	informerspolicynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/k8us/v1"
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
	quotas map[string]*core_v1.ResourceQuota
	// Parent pointers from one namespace to another
	parents map[string]string
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
	resourceQuotas, err := c.resourceQuotaInformer.Lister().List(labels.Everything())
	if err != nil {
		return err
	}
	policyNodes, err := c.policyNodeInformer.Lister().List(labels.Everything())
	if err != nil {
		return err
	}
	c.parents = map[string]string{}
	c.quotas = map[string]*core_v1.ResourceQuota{}

	for _, policyNode := range policyNodes {
		c.parents[policyNode.Name] = policyNode.Spec.Parent

		c.quotas[policyNode.Name] = &core_v1.ResourceQuota{
			Spec: *policyNode.Spec.Policies.ResourceQuota.DeepCopy(),
			Status: core_v1.ResourceQuotaStatus{
				Used: core_v1.ResourceList{},
			},
		}
	}

	// Set the usage based on the quota informer
	for _, resourceQuota := range resourceQuotas {
		if resourceQuota.Name != ResourceQuotaObjectName {
			continue // Only care about stolos resource quota objects
		}

		quota, exists := c.quotas[resourceQuota.Namespace]
		if !exists {
			glog.Infof("Resource Quota exists for namespace %s not defined in policy nodes", resourceQuota.Namespace)
			continue // This can happen frequently during deletions and while adjusting the tree.
		}
		// For leaf
		resourceQuota.Status.DeepCopyInto(&quota.Status)

		// For all the parents, add up quantities
		parent := c.parents[resourceQuota.Namespace]
		for parent != pn_v1.NoParentNamespace {
			quota, exists := c.quotas[parent]
			if !exists {
				glog.Warningf("Parent namespace %s not defined in policy nodes for child namespace %s",
					parent, resourceQuota.Namespace)
				break
			}
			for resourceName, quantity := range resourceQuota.Status.Used {
				if current, exists := quota.Status.Used[resourceName]; exists {
					current.Add(quantity)
					quota.Status.Used[resourceName] = current
				} else {
					quota.Status.Used[resourceName] = quantity
				}
			}
			parent = c.parents[parent]
		}
	}
	return nil
}

// admit checks whether the new usage can be applied to the provided namespace and its ancestors.
// If cannot admit returns an error describing the quota that was violated.
func (c *HierarchicalQuotaCache) Admit(namespace string, newUsageList core_v1.ResourceList) error {
	for namespace != pn_v1.NoParentNamespace {
		namespaceQuota, exists := c.quotas[namespace]
		if !exists {
			// No namespace defined in policy nodes so this is not a namespace controlled by stolos.
			return nil
		}
		for resourceName, newUsage := range newUsageList {
			current, exists := namespaceQuota.Status.Used[resourceName]
			if !exists {
				current = resource.MustParse("0")
			}
			limit, exists := namespaceQuota.Spec.Hard[resourceName]
			if exists {
				newTotalUsage := current.Copy()
				newTotalUsage.Add(newUsage)
				if newTotalUsage.Cmp(limit) > 0 {
					return errors.Errorf("exceeded quota in namespace %s, requested: %s=%d, limit: %s=%d",
						namespace, resourceName, newTotalUsage.Value(), resourceName, limit.Value())
				}
			}
		}
		namespace = c.parents[namespace]
	}
	return nil
}

// Collect all the limits (max for each) that are set above this leaf quota. This should not be used to make admission
// decisions. See Quota Syncer for usage.
func (c *HierarchicalQuotaCache) GetParentQuotaLimits(namespace string) core_v1.ResourceList {
	parentQuotaLimits := core_v1.ResourceList{}

	parentNamespace := c.parents[namespace]
	for parentNamespace != pn_v1.NoParentNamespace {
		parentSpec, exists := c.quotas[parentNamespace]
		if !exists {
			glog.Infof("Parent %s does not have a policyNode", parentNamespace)
			break
		}
		for resource, limit := range parentSpec.Spec.Hard {
			if limitSoFar, exists := parentQuotaLimits[resource]; exists {
				if limitSoFar.Cmp(limit) < 0 {
					parentQuotaLimits[resource] = limit
				}
			} else {
				parentQuotaLimits[resource] = limit
			}
		}
		parentNamespace = c.parents[parentNamespace]
	}

	return parentQuotaLimits
}
