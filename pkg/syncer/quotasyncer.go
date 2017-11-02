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

package syncer

import (
	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	informers_policynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/k8us/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/resource-quota"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/pkg/errors"
	informers_corev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"
)

// QuotaSyncer handles syncing quota from PolicyNodes.
type QuotaSyncer struct {
	client                kubernetes.Interface
	resourceQuotaInformer informers_corev1.ResourceQuotaInformer
	resourceQuotaLister   listers_core_v1.ResourceQuotaLister
	policyNodeInformer    informers_policynodev1.PolicyNodeInformer
	queue                 workqueue.RateLimitingInterface
}

var _ PolicyNodeSyncerInterface = &QuotaSyncer{}

// NewQuotaSyncer creates a quota syncer that will use the given client.
func NewQuotaSyncer(
	client meta.Interface,
	resourceQuotaInformer informers_corev1.ResourceQuotaInformer,
	policyNodeInformer informers_policynodev1.PolicyNodeInformer,
	queue workqueue.RateLimitingInterface) *QuotaSyncer {
	return &QuotaSyncer{
		client:                client.Kubernetes(),
		resourceQuotaInformer: resourceQuotaInformer,
		resourceQuotaLister:   resourceQuotaInformer.Lister(),
		policyNodeInformer:    policyNodeInformer,
		queue:                 queue,
	}
}

// PeriodicResync implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) PeriodicResync(nodes []*policyhierarchy_v1.PolicyNode) error {
	actionsToQueue, err := s.fillResourceQuotaLeafGaps(nodes)
	if err != nil {
		return err
	}

	for _, action := range actionsToQueue {
		s.queue.Add(action)
	}

	return nil
}

// PeriodicResync is used here to "fill in the gaps" for ResourceQuota in leaf nodes.
// Stolos ResourceQuota relies on the native Kubernetes resource quota controllers to monitor usage,
// But the native controllers don't monitor usage unless a limit is set.
// This means that if a parent policyspace has a limit set, we need to make sure that the leaf namespace
// also has that resource type limited, so that we get usage stats from it to allow enforcement of that limit.
func (s *QuotaSyncer) fillResourceQuotaLeafGaps(nodes []*policyhierarchy_v1.PolicyNode) ([]actions.ResourceQuotaAction, error) {
	resultActions := []actions.ResourceQuotaAction{}

	cache, err := resource_quota.NewHierarchicalQuotaCache(s.policyNodeInformer, s.resourceQuotaInformer)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed constructing hierarchical quota cache during periodic resync")
	}

	for _, node := range nodes {
		if node.Spec.Policyspace {
			continue
		}

		// Get the limits that are set above this leaf quota in the policyspace hierarchy
		parentQuotaLimits := cache.GetParentQuotaLimits(node.Name)
		quota, err := s.resourceQuotaInformer.Lister().ResourceQuotas(node.Name).Get(resource_quota.ResourceQuotaObjectName)
		if err != nil {
			glog.Infof("Error getting quota object for leaf namespace %s during resync, continuing", node.Name)
			continue
		}

		// If the limits above are not all contained in this quota, we need to update the quota object
		// with extra limits so that the native controller tracks those resource types
		needsUpdate := false
		for resource, limit := range parentQuotaLimits {
			if _, exists := quota.Spec.Hard[resource]; !exists {
				// NOTE: Soon a feature will come that will let us set it in "soft" instead of "hard".
				// Switch to that when available.
				quota.Spec.Hard[resource] = limit
				needsUpdate = true
			}
		}

		if needsUpdate {
			glog.Infof("Need to update quota for leaf %s to fill in limits from parent policyspaces", quota.Namespace)
			resultActions = append(resultActions, actions.NewResourceQuotaUpsertAction(
				quota.Namespace, resource_quota.StolosQuotaLabels, quota.Spec, s.client, s.resourceQuotaLister))
		}
	}
	return resultActions, nil
}

// OnCreate implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnCreate(policyNode *policyhierarchy_v1.PolicyNode) error {
	return s.onUpdate(policyNode)
}

// getUpdateAction returns the appropraite action when handling an update event.
func (s *QuotaSyncer) getUpdateAction(policyNode *policyhierarchy_v1.PolicyNode) actions.ResourceQuotaAction {
	if policyNode.Spec.Policyspace {
		return actions.NewResourceQuotaUpsertAction(
			policyNode.Namespace,
			resource_quota.PolicySpaceQuotaLabels,
			policyNode.Spec.Policies.ResourceQuota,
			s.client,
			s.resourceQuotaLister)
	}

	// TODO(mdruskin): have this evaluate hierarchical policy instead of this.
	if 0 < len(policyNode.Spec.Policies.ResourceQuota.Hard) {
		return actions.NewResourceQuotaUpsertAction(
			policyNode.Name,
			resource_quota.PolicySpaceQuotaLabels,
			policyNode.Spec.Policies.ResourceQuota,
			s.client,
			s.resourceQuotaLister)
	}
	return actions.NewResourceQuotaDeleteAction(
		policyNode.Name,
		s.client,
		s.resourceQuotaLister)
}

// OnUpdate implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnUpdate(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	return s.onUpdate(new)
}

// onUpdate handles both create and update for quota from a policy node.
func (s *QuotaSyncer) onUpdate(policyNode *policyhierarchy_v1.PolicyNode) error {
	s.queue.Add(s.getUpdateAction(policyNode))
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	glog.Infof("Got deleted policy node event %s, ignoring since the resource quota will be auto-deleted", node.Name)
	return nil
}
