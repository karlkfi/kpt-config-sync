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
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/syncer/labeling"
	"github.com/google/nomos/pkg/util/objectreflection"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterPolicyUnpackerInterface defines the interface for transforming ClusterPolicy into resources
// as well as listing the currently instantiated resources from the cluster scope.
type ClusterPolicyUnpackerInterface interface {
	// UpdateRemovals returns a list of pointers to resource to delete.
	// The first arg is the previous value for the ClusterPolicy, and the second is the new value.
	// Note that the informer framework will periodically re-list and pass all existing clusterPolicys as
	// an "update" where old and new both have identical resource versions values.
	UpdateRemovals(clusterPolicy *policyhierarchy_v1.ClusterPolicy, new *policyhierarchy_v1.ClusterPolicy) []runtime.Object

	// Upserts takes a ClusterPolicy, and transforms it into a list of pointers to resources that are declared
	// in the ClusterPolicy object.
	Upserts(clusterPolicy *policyhierarchy_v1.ClusterPolicy) []runtime.Object

	// Names takes a ClusterPolicy and transforms it into names of the resources that are declared in the
	// ClusterPolicy. For each resource name, the function will set a key-value of (resource name, true)
	// in the returned map.
	Names(clusterPolicy *policyhierarchy_v1.ClusterPolicy) map[string]bool

	// List returns all items on the cluster from a lister.
	List() ([]runtime.Object, error)

	// NewDeleteAction creates a new delete action
	NewDeleteAction(name string) action.Interface

	// NewUpsertAction creates a new upsert action
	NewUpsertAction(name string, obj runtime.Object) action.Interface
}

// Enqueuer implements the "Add" method of workqueue.RateLimitingInterface
type Enqueuer interface {
	Add(item interface{})
}

// ClusterGenericSyncer will sync cluster scoped resources that are defined in ClusterPolicy
type ClusterGenericSyncer struct {
	queue            Enqueuer                       // Queue for created operations
	resourceUnpacker ClusterPolicyUnpackerInterface // Interface for syncing the resoruce from ClusterPolicy
}

// ClusterGenericSyncer implements ClusterPolicySyncerInterface
var _ ClusterPolicySyncerInterface = &ClusterGenericSyncer{}

// NewClusterGenericSyncer creates a generic cluster scoped syncer object which will operate on
// a type defined by ClusterPolicyUnpackerInterface.
func NewClusterGenericSyncer(
	resourceUnpacker ClusterPolicyUnpackerInterface,
	queue Enqueuer) *ClusterGenericSyncer {
	return &ClusterGenericSyncer{
		resourceUnpacker: resourceUnpacker,
		queue:            queue,
	}
}

// OnCreate implements ClusterPolicySyncerInterface
func (s *ClusterGenericSyncer) OnCreate(clusterPolicy *policyhierarchy_v1.ClusterPolicy) error {
	if err := s.handleNotFound(clusterPolicy); err != nil {
		return err
	}
	s.onSet(clusterPolicy)
	return nil
}

// OnUpdate implements ClusterPolicySyncerInterface
func (s *ClusterGenericSyncer) OnUpdate(
	oldClusterPolicy *policyhierarchy_v1.ClusterPolicy, clusterPolicy *policyhierarchy_v1.ClusterPolicy) error {
	if err := s.handleNotFound(clusterPolicy); err != nil {
		return err
	}
	s.handleRemovals(oldClusterPolicy, clusterPolicy)
	s.onSet(clusterPolicy)
	return nil
}

func (s *ClusterGenericSyncer) onSet(clusterPolicy *policyhierarchy_v1.ClusterPolicy) {
	for _, item := range s.resourceUnpacker.Upserts(clusterPolicy) {
		_, name := objectreflection.GetNamespaceAndName(item)
		s.queue.Add(s.resourceUnpacker.NewUpsertAction(name, item))
	}
}

func (s *ClusterGenericSyncer) handleRemovals(
	old *policyhierarchy_v1.ClusterPolicy, new *policyhierarchy_v1.ClusterPolicy) {
	for _, item := range s.resourceUnpacker.UpdateRemovals(old, new) {
		_, name := objectreflection.GetNamespaceAndName(item)
		s.queue.Add(s.resourceUnpacker.NewDeleteAction(name))
	}
}

func (s *ClusterGenericSyncer) handleNotFound(declared *policyhierarchy_v1.ClusterPolicy) error {
	names := s.resourceUnpacker.Names(declared)
	items, err := s.resourceUnpacker.List()
	if err != nil {
		return err
	}

	for _, item := range items {
		_, objectMeta := objectreflection.Meta(item)
		if !names[objectMeta.Name] && labeling.HasOriginLabel(*objectMeta) {
			s.queue.Add(s.resourceUnpacker.NewDeleteAction(objectMeta.Name))
		}
	}
	return nil
}

// OnDelete implements ClusterPolicySyncerInterface
func (s *ClusterGenericSyncer) OnDelete(clusterPolicy *policyhierarchy_v1.ClusterPolicy) error {
	if err := s.handleNotFound(clusterPolicy); err != nil {
		return err
	}
	s.handleRemovals(clusterPolicy, &policyhierarchy_v1.ClusterPolicy{})
	return nil
}
