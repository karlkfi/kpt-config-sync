/*
Copyright 2017 The Nomos Authors.
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
	"github.com/google/nomos/pkg/syncer/actions"
)

// UnpackerInterface defines the interface for transforming policy nodes into resources as well as
// listing the currently instantiated resources from a namespace.
type UnpackerInterface interface {
	// UpdateRemovals returns a list of pointers to resource to delete.
	// The first arg is the previous value for the policy node, and the second is the new value.
	// Note that the informer framework will periodically re-list and pass all existing nodes as
	// an "update" where old and new both have identical resource versions values.
	UpdateRemovals(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) []interface{}

	// Upserts takes a node, and transforms it into a list of pointers to resources that are declared
	// in the PolicyNode object.
	Upserts(node *policyhierarchy_v1.PolicyNode) []interface{}

	// Names takes a PolicyNode and transforms it into names of the resources that are declared in the
	// PolicyNode. For each resource name, the function will set a key-value of (resource name, true)
	// in the returned map.
	Names(node *policyhierarchy_v1.PolicyNode) map[string]bool
}

// Enqueuer implements the "Add" method of workqueue.RateLimitingInterface
type Enqueuer interface {
	Add(item interface{})
}

// GenericSyncer will sync namespaced resources that are defined in policy nodes
type GenericSyncer struct {
	queue            Enqueuer                  // Queue for created operations
	resourceAction   actions.ResourceInterface // Interface for performing actions
	resourceUnpacker UnpackerInterface         // Interface for syncing the resoruce from policy nodes
}

// GenericSyncer implements PolicyNodeSyncerInterface
var _ PolicyNodeSyncerInterface = &GenericSyncer{}

// NewGenericSyncer creates the RBAC syncer object
func NewGenericSyncer(
	resourceAction actions.ResourceInterface,
	resourceUnpacker UnpackerInterface,
	queue Enqueuer) *GenericSyncer {
	return &GenericSyncer{
		resourceAction:   resourceAction,
		resourceUnpacker: resourceUnpacker,
		queue:            queue,
	}
}

// OnCreate implements PolicyNodeSyncerInterface
func (s *GenericSyncer) OnCreate(node *policyhierarchy_v1.PolicyNode) error {
	s.onSet(node)
	return nil
}

// OnUpdate implements PolicyNodeSyncerInterface
func (s *GenericSyncer) OnUpdate(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	for _, removeItem := range s.resourceUnpacker.UpdateRemovals(old, new) {
		s.queue.Add(actions.NewGenericDeleteAction(removeItem, s.resourceAction))
	}
	s.onSet(new)
	return nil
}

func (s *GenericSyncer) onSet(node *policyhierarchy_v1.PolicyNode) {
	for _, item := range s.resourceUnpacker.Upserts(node) {
		s.queue.Add(actions.NewGenericUpsertAction(item, s.resourceAction))
	}
}

// OnDelete implements PolicyNodeSyncerInterface
func (s *GenericSyncer) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	// Resource will be deleted by namespace deletion.
	return nil
}

// PeriodicResync implements PolicyNodeSyncerInterface
func (s *GenericSyncer) PeriodicResync(nodes []*policyhierarchy_v1.PolicyNode) error {
	for _, node := range nodes {
		specified := s.resourceUnpacker.Names(node)
		items, err := s.resourceAction.Values(node.Name)
		if err != nil {
			return err
		}
		for name, item := range items {
			if !specified[name] {
				s.queue.Add(actions.NewGenericDeleteAction(item, s.resourceAction))
			}
		}
	}
	return nil
}
