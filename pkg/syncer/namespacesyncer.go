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
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/google/stolos/pkg/util/namespaceutil"
	"github.com/google/stolos/pkg/util/set/stringset"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"
)

// NamespaceSyncer handles syncing namespaces from policy nodes.
type NamespaceSyncer struct {
	client          meta.Interface
	namespaceLister listers_core_v1.NamespaceLister
	queue           workqueue.RateLimitingInterface
}

var _ PolicyNodeSyncerInterface = &NamespaceSyncer{}

// NewNamespaceSyncer creates a new namespace syncer from the client.
func NewNamespaceSyncer(
	client meta.Interface,
	namespaceLister listers_core_v1.NamespaceLister,
	queue workqueue.RateLimitingInterface) *NamespaceSyncer {
	return &NamespaceSyncer{
		client:          client,
		namespaceLister: namespaceLister,
		queue:           queue,
	}
}

// OnCreate implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnCreate(node *policyhierarchy_v1.PolicyNode) error {
	return s.onUpsert(node)
}

// OnUpdate implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnUpdate(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	return s.onUpsert(new)
}

func (s *NamespaceSyncer) onUpsert(node *policyhierarchy_v1.PolicyNode) error {
	s.queue.Add(actions.NewNamespaceUpsertAction(
		node.Name,
		map[string]string{policyhierarchy_v1.ParentLabelKey: node.Spec.Parent},
		s.client.Kubernetes(),
		s.namespaceLister))
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	s.queue.Add(actions.NewNamespaceDeleteAction(node.Name, s.client.Kubernetes(), s.namespaceLister))
	return nil
}

// PeriodicResync implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) PeriodicResync(nodes []*policyhierarchy_v1.PolicyNode) error {
	existingNamespaces, err := s.namespaceLister.List(labels.Everything())
	if err != nil {
		return err
	}

	namespaceActions := s.computeActions(existingNamespaces, nodes)
	for _, action := range namespaceActions {
		s.queue.Add(action)
	}
	return nil
}

// computeActions determines which namespaces to delete during the resync. Creates will be handled
// by OnUpdate since every resource is "updated" during the resync. Deletes are handled by OnDelete
// but if we miss a delete due to being off, crashed, etc, this will garbage collect ones that
// we missed.
func (s *NamespaceSyncer) computeActions(
	existingNamespaceList []*core_v1.Namespace,
	nodes []*policyhierarchy_v1.PolicyNode) []actions.Interface {
	existingNamespaces := stringset.New()
	for _, namespace := range existingNamespaceList {
		if namespaceutil.IsReserved(*namespace) {
			continue
		}
		switch namespace.Status.Phase {
		case core_v1.NamespaceActive:
			existingNamespaces.Add(namespace.Name)
		case core_v1.NamespaceTerminating:
			// noop since this will go away shortly
		}
	}

	declaredNamespaces := stringset.New()
	for _, policyNode := range nodes {
		declaredNamespaces.Add(policyNode.ObjectMeta.Name)
	}

	needsDelete := existingNamespaces.Difference(declaredNamespaces)

	namespaceActions := []actions.Interface{}
	needsDelete.ForEach(func(ns string) {
		glog.Infof("Adding delete operation for %s", ns)
		namespaceActions = append(namespaceActions, actions.NewNamespaceDeleteAction(
			ns, s.client.Kubernetes(), s.namespaceLister))
	})

	return namespaceActions
}
