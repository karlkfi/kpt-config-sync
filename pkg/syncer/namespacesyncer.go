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
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/syncer/actions"
	"github.com/google/nomos/pkg/syncer/labeling"
	"k8s.io/apimachinery/pkg/labels"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"
)

// NamespaceSyncer handles syncing namespaces from policy nodes.
type NamespaceSyncer struct {
	client            meta.Interface
	namespaceLister   listers_core_v1.NamespaceLister
	namespaceSelector labels.Selector
	queue             workqueue.RateLimitingInterface
}

var _ PolicyNodeSyncerInterface = &NamespaceSyncer{}

// NewNamespaceSyncer creates a new namespace syncer from the client.
func NewNamespaceSyncer(
	client meta.Interface,
	namespaceLister listers_core_v1.NamespaceLister,
	queue workqueue.RateLimitingInterface) *NamespaceSyncer {
	return &NamespaceSyncer{
		client:            client,
		namespaceLister:   namespaceLister,
		namespaceSelector: labeling.NewOriginSelector(),
		queue:             queue,
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
	if !node.Spec.Policyspace {
		s.queue.Add(actions.NewNamespaceUpsertAction(
			node.Name,
			node.UID,
			map[string]string{policyhierarchy_v1.ParentLabelKey: node.Spec.Parent},
			s.client.Kubernetes(),
			s.namespaceLister))
	}
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	// Can be ignored, garbage collector will handle this case.
	return nil
}

// PeriodicResync implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) PeriodicResync(nodes []*policyhierarchy_v1.PolicyNode) error {
	// TODO: delete this.
	return nil
}
