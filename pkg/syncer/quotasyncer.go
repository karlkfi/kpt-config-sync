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

package syncer

import (
	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	informers_policynodev1 "github.com/google/nomos/pkg/client/informers/externalversions/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/syncer/actions"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
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

	for _, node := range nodes {
		if node.Spec.Policyspace {
			continue
		}

		// Get the limits that are set above this leaf quota in the policyspace hierarchy
		parentQuotaLimits := s.getHierarchicalQuotaLimits(*node)

		quota, err := s.resourceQuotaInformer.Lister().ResourceQuotas(node.Name).Get(resourcequota.ResourceQuotaObjectName)
		if err != nil {
			if api_errors.IsNotFound(err) {
				quota = &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{},
					},
				}
			} else {
				glog.Warningf(
					"Failed to get quota object for leaf namespace %s during resync, continuing: %v",
					node.Name, err)
				continue
			}
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
			glog.Infof("Need to update quota for leaf %s to fill in limits from parent policyspaces", node.Name)
			resultActions = append(resultActions, actions.NewResourceQuotaUpsertAction(
				node.Name, resourcequota.StolosQuotaLabels, quota.Spec, s.client, s.resourceQuotaLister))
		}
	}
	return resultActions, nil
}

// OnCreate implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnCreate(policyNode *policyhierarchy_v1.PolicyNode) error {
	return s.onUpdate(policyNode)
}

// getUpdateAction returns the appropriate action when handling an update event.
func (s *QuotaSyncer) getUpdateAction(policyNode *policyhierarchy_v1.PolicyNode) actions.ResourceQuotaAction {
	if policyNode.Spec.Policyspace {
		return nil
	}

	hierarchicalLimits := s.getHierarchicalQuotaLimits(*policyNode)

	if len(hierarchicalLimits) > 0 {
		return actions.NewResourceQuotaUpsertAction(
			policyNode.Name,
			resourcequota.StolosQuotaLabels,
			core_v1.ResourceQuotaSpec{Hard: hierarchicalLimits},
			s.client,
			s.resourceQuotaLister)
	}
	return actions.NewResourceQuotaDeleteAction(
		policyNode.Name,
		s.client,
		s.resourceQuotaLister)
}

// getHierarchicalQuotaLimits takes in the limits of a policyNode and adds in the limits of the node's ancestors if
// the limits are not already present. This is needed to ensure that the native quota controllers monitor usage of
// resources for quotas only defined in the parent nodes.
func (q *QuotaSyncer) getHierarchicalQuotaLimits(policyNode policyhierarchy_v1.PolicyNode) core_v1.ResourceList {
	var parentNamespace string
	var err error
	hierarchicalLimits := core_v1.ResourceList{}

	for parentNode := &policyNode; err == nil; parentNode, err = q.policyNodeInformer.Lister().Get(parentNamespace) {
		if parentNode.Spec.Policies.ResourceQuotaV1 != nil {
			for resource, limit := range parentNode.Spec.Policies.ResourceQuotaV1.Spec.Hard {
				if _, exists := hierarchicalLimits[resource]; !exists {
					hierarchicalLimits[resource] = limit
				}
			}
		}
		parentNamespace = parentNode.Spec.Parent
	}
	return hierarchicalLimits
}

// OnUpdate implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnUpdate(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	return s.onUpdate(new)
}

// onUpdate handles both create and update for quota from a policy node.
func (s *QuotaSyncer) onUpdate(policyNode *policyhierarchy_v1.PolicyNode) error {
	action := s.getUpdateAction(policyNode)
	if action != nil {
		s.queue.Add(action)
	}
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	glog.Infof("Got deleted policy node event %s, ignoring since the resource quota will be auto-deleted", node.Name)
	return nil
}
