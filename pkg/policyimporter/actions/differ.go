/*
Copyright 2018 The Kubernetes Authors.
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

package actions

import (
	"flag"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/policyimporter"
	"github.com/google/nomos/pkg/util/policynode"
)

var syncDeleteMaxWait = flag.Duration("sync_delete_max_wait", 30*time.Second,
	"Number of seconds to wait for Syncer to acknowledge Sync deletion")

// Differ will generate an ordered list of actions needed to transition policy from the current to
// desired state.
//
// Maintains the invariant that the policy node tree is valid (i.e. no cycles, parents pointing to existing nodes),
// assuming the current and desired state are valid themselves.
//
// More details about the algorithm can be found at docs/update-preserving-invariants.md
type Differ struct {
	factories Factories
	// SortDiff is true when we enforce an ordering for iterating over PolicyNode maps.
	SortDiff bool
}

// NewDiffer creates a Differ.
func NewDiffer(factories Factories) *Differ {
	return &Differ{factories: factories}
}

// Diff returns a list of actions that when applied, transitions the current state to desired state.
// Note that the invariants are only maintained if the actions are processed by a single thread in order.
//
// This list does not include Sync delete actions, because those are handled differently. The caller
// must call SyncDeletes and process those actions before processing these actions.
func (d *Differ) Diff(current, desired policynode.AllPolicies) []action.Interface {
	var actions []action.Interface
	actions = append(actions, d.policyNodeActions(current, desired)...)
	actions = append(actions, d.clusterPolicyActions(current, desired)...)
	actions = append(actions, d.syncActions(current, desired)...)
	return actions
}

func (d *Differ) policyNodeActions(current, desired policynode.AllPolicies) []action.Interface {
	var actions []action.Interface
	var deletes, creates, updates int
	for name := range desired.PolicyNodes {
		intent := desired.PolicyNodes[name]
		if actual, found := current.PolicyNodes[name]; found {
			if !d.factories.PolicyNodeAction.Equal(&intent, &actual) {
				actions = append(actions, d.factories.PolicyNodeAction.NewUpdate(&intent))
				updates++
			}
		} else {
			actions = append(actions, d.factories.PolicyNodeAction.NewCreate(&intent))
			creates++
		}
	}
	for name := range current.PolicyNodes {
		if _, found := desired.PolicyNodes[name]; !found {
			actions = append(actions, d.factories.PolicyNodeAction.NewDelete(name))
			deletes++
		}
	}

	glog.Infof("PolicyNode operations: create %d, update %d, delete %d", creates, updates, deletes)
	policyimporter.Metrics.Operations.WithLabelValues("create").Add(float64(creates))
	policyimporter.Metrics.Operations.WithLabelValues("update").Add(float64(updates))
	policyimporter.Metrics.Operations.WithLabelValues("delete").Add(float64(deletes))
	return actions
}

func (d *Differ) clusterPolicyActions(current, desired policynode.AllPolicies) []action.Interface {
	var actions []action.Interface
	if current.ClusterPolicy == nil && desired.ClusterPolicy == nil {
		return actions
	}
	if current.ClusterPolicy == nil {
		actions = []action.Interface{d.factories.ClusterPolicyAction.NewCreate(desired.ClusterPolicy)}
	} else if desired.ClusterPolicy == nil {
		actions = []action.Interface{d.factories.ClusterPolicyAction.NewDelete(current.ClusterPolicy.Name)}
	} else if !d.factories.ClusterPolicyAction.Equal(desired.ClusterPolicy, current.ClusterPolicy) {
		actions = []action.Interface{d.factories.ClusterPolicyAction.NewUpdate(desired.ClusterPolicy)}
	}
	return actions
}

func (d *Differ) syncActions(current, desired policynode.AllPolicies) []action.Interface {
	var actions []action.Interface
	var creates, updates, deletes int
	for name, newSync := range desired.Syncs {
		if oldSync, exists := current.Syncs[name]; exists {
			if !d.factories.SyncAction.Equal(&newSync, &oldSync) {
				actions = append(actions, d.factories.SyncAction.NewUpdate(newSync))
				updates++
			}
		} else {
			actions = append(actions, d.factories.SyncAction.NewCreate(newSync))
			creates++
		}
	}

	for name := range current.Syncs {
		if _, found := desired.Syncs[name]; !found {
			actions = append(actions, d.factories.SyncAction.NewDelete(name, *syncDeleteMaxWait))
			deletes++
		}
	}

	glog.Infof("Sync operations: %d updates, %d creates, %d deletes", updates, creates, deletes)
	return actions
}

// SyncDeletes returns a list of names of Syncs to be deleted.
// The caller must delete all these syncs and wait for them to be finalized before processing other
// actions.
func (d *Differ) SyncDeletes(current, desired map[string]v1alpha1.Sync) []string {
	var toDelete []string
	for _, sync := range current {
		if _, exists := desired[sync.Name]; !exists {
			toDelete = append(toDelete, sync.Name)
		}
	}
	glog.Infof("Sync operations: %d deletes", len(toDelete))
	return toDelete
}
