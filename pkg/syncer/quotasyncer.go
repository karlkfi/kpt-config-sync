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
	"reflect"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	informers_policynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/k8us/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/resource-quota"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	informers_corev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
)

// QuotaSyncer handles syncing quota from PolicyNodes.
type QuotaSyncer struct {
	client                kubernetes.Interface
	resourceQuotaInformer informers_corev1.ResourceQuotaInformer
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

	// Need to call this so that it gets populated.
	resourceQuotaInformer.Informer()

	return &QuotaSyncer{
		client:                client.Kubernetes(),
		resourceQuotaInformer: resourceQuotaInformer,
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
			resultActions = append(resultActions,
				actions.NewResourceQuotaUpdateAction(s.client, quota.Namespace, quota.Spec, quota.ResourceVersion))
		}
	}
	return resultActions, nil
}

// OnCreate implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnCreate(policyNode *policyhierarchy_v1.PolicyNode) error {
	return s.onUpdate(policyNode)
}

// getUpdateAction returns the appropraite action when handling an update event.
func (s *QuotaSyncer) getUpdateAction(
	policyNode *policyhierarchy_v1.PolicyNode) (actions.ResourceQuotaAction, error) {
	namespace := policyNode.Name
	// NOTE: Get will return a non-nil ResourceQutoa even if the API returns not found.
	existingResourceQuota, err := s.resourceQuotaInformer.Lister().ResourceQuotas(namespace).Get(
		resource_quota.ResourceQuotaObjectName)
	hasExistingResourceQuota := true
	if err != nil {
		if api_errors.IsNotFound(err) {
			hasExistingResourceQuota = false
			existingResourceQuota = nil
		} else {
			return nil, errors.Wrapf(err, "Failed to fetch quota for %s", namespace)
		}
	}

	var neededResourceQuotaSpec *core_v1.ResourceQuotaSpec
	if !policyNode.Spec.Policyspace && len(policyNode.Spec.Policies.ResourceQuota.Hard) > 0 {
		neededResourceQuotaSpec = &policyNode.Spec.Policies.ResourceQuota
	}
	if !hasExistingResourceQuota && neededResourceQuotaSpec != nil {
		return actions.NewResourceQuotaCreateAction(s.client, namespace, *neededResourceQuotaSpec), nil
	}
	if hasExistingResourceQuota && neededResourceQuotaSpec == nil {
		glog.V(1).Infof("Will delete ns %s quota %#v", namespace, existingResourceQuota)
		return actions.NewResourceQuotaDeleteAction(s.client, namespace), nil
	}
	if hasExistingResourceQuota && neededResourceQuotaSpec != nil &&
		!reflect.DeepEqual(existingResourceQuota.Spec, *neededResourceQuotaSpec) {
		return actions.NewResourceQuotaUpdateAction(
			s.client, namespace, *neededResourceQuotaSpec, existingResourceQuota.ObjectMeta.ResourceVersion), nil
	}
	return nil, nil
}

// OnUpdate implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnUpdate(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	return s.onUpdate(new)
}

// onUpdate handles both create and update for quota from a policy node.
func (s *QuotaSyncer) onUpdate(policyNode *policyhierarchy_v1.PolicyNode) error {
	action, err := s.getUpdateAction(policyNode)
	if err != nil {
		return err
	}
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
