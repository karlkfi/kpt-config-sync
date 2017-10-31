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
	"github.com/google/stolos/pkg/client/meta"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/google/stolos/pkg/resource-quota"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/workqueue"
)

// QuotaSyncer handles syncing quota from PolicyNodes.
type QuotaSyncer struct {
	client              kubernetes.Interface
	resourceQuotaLister listers_core_v1.ResourceQuotaLister
	queue               workqueue.RateLimitingInterface
}

var _ PolicyNodeSyncerInterface = &QuotaSyncer{}

// NewQuotaSyncer creates a quota syncer that will use the given client.
func NewQuotaSyncer(
	client meta.Interface,
	resourceQuotaLister listers_core_v1.ResourceQuotaLister,
	queue workqueue.RateLimitingInterface) *QuotaSyncer {
	return &QuotaSyncer{
		client:              client.Kubernetes(),
		resourceQuotaLister: resourceQuotaLister,
		queue:               queue,
	}
}

// InitialSync implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) InitialSync(nodes []*policyhierarchy_v1.PolicyNode) error {

	resourceQuotaList, err := s.resourceQuotaLister.ResourceQuotas(meta_v1.NamespaceAll).List(
		labels.SelectorFromSet(resource_quota.StolosQuotaLabels))
	if err != nil {
		return err
	}

	actions := s.computeActions(resourceQuotaList, nodes)

	for _, action := range actions {
		s.queue.Add(action)
	}

	return nil
}

func (s *QuotaSyncer) computeActions(
	resourceQuotaList []*core_v1.ResourceQuota, policyNodes []*policyhierarchy_v1.PolicyNode) []actions.ResourceQuotaAction {
	existing := map[string]core_v1.ResourceQuota{}
	for _, rq := range resourceQuotaList {
		existing[rq.Namespace] = *rq
	}

	declaring := map[string]core_v1.ResourceQuotaSpec{}
	for _, pn := range policyNodes {
		// TODO(mdruskin): If not working namespace, we should create a hierarchical resource quota
		if !pn.Spec.Policyspace {
			declaring[pn.Name] = pn.Spec.Policies.ResourceQuota
		}
	}

	quotaActions := []actions.ResourceQuotaAction{}
	// Creates and updates
	for ns, rq := range declaring {
		if _, exists := existing[ns]; !exists {
			quotaActions = append(quotaActions, actions.NewResourceQuotaCreateAction(s.client, ns, rq))
		} else if !reflect.DeepEqual(rq, existing[ns].Spec) {
			quotaActions = append(quotaActions, actions.NewResourceQuotaUpdateAction(s.client, ns, rq, existing[ns].ResourceVersion))
		}
	}
	// Deletions
	for ns, _ := range existing {
		if _, exists := declaring[ns]; !exists {
			quotaActions = append(quotaActions, actions.NewResourceQuotaDeleteAction(s.client, ns))
		}
	}

	return quotaActions
}

func (s *QuotaSyncer) getCreateAction(policyNode *policyhierarchy_v1.PolicyNode) actions.ResourceQuotaAction {
	if policyNode.Spec.Policyspace || len(policyNode.Spec.Policies.ResourceQuota.Hard) == 0 {
		return nil
	}
	return actions.NewResourceQuotaCreateAction(
		s.client, policyNode.Name, policyNode.Spec.Policies.ResourceQuota)
}

// OnCreate implements PolicyNodeSyncerInterface
func (s *QuotaSyncer) OnCreate(policyNode *policyhierarchy_v1.PolicyNode) error {
	if action := s.getCreateAction(policyNode); action != nil {
		s.queue.Add(action)
	}
	return nil
}

// getUpdateAction returns the appropraite action when handling an update event.
func (s *QuotaSyncer) getUpdateAction(
	policyNode *policyhierarchy_v1.PolicyNode) (actions.ResourceQuotaAction, error) {
	namespace := policyNode.Name
	// NOTE: Get will return a non-nil ResourceQutoa even if the API returns not found.
	existingResourceQuota, err := s.resourceQuotaLister.ResourceQuotas(namespace).Get(
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
func (s *QuotaSyncer) OnUpdate(old *policyhierarchy_v1.PolicyNode, newPolicyNode *policyhierarchy_v1.PolicyNode) error {
	action, err := s.getUpdateAction(newPolicyNode)
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
