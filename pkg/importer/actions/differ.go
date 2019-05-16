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
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/util/namespaceconfig"
)

var syncDeleteMaxWait = flag.Duration("sync_delete_max_wait", 30*time.Second,
	"Number of seconds to wait for Syncer to acknowledge Sync deletion")

// Differ will generate an ordered list of actions needed to transition config from the current to
// desired state.
//
// Maintains the invariant that the config tree is valid (i.e. no cycles, parents pointing to
// existing configs), assuming the current and desired state are valid themselves.
//
// More details about the algorithm can be found at docs/update-preserving-invariants.md
type Differ struct {
	factories Factories
	// SortDiff is true when we enforce an ordering for iterating over NamespaceConfig maps.
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
func (d *Differ) Diff(current, desired namespaceconfig.AllConfigs) []action.Interface {
	var actions []action.Interface
	actions = append(actions, d.namespaceConfigActions(current, desired)...)
	actions = append(actions, d.clusterConfigActions(current, desired)...)
	actions = append(actions, d.syncActions(current, desired)...)
	return actions
}

func (d *Differ) namespaceConfigActions(current, desired namespaceconfig.AllConfigs) []action.Interface {
	var actions []action.Interface
	var deletes, creates, updates int
	for name := range desired.NamespaceConfigs {
		intent := desired.NamespaceConfigs[name]
		if actual, found := current.NamespaceConfigs[name]; found {
			if !d.factories.NamespaceConfigAction.Equal(&intent, &actual) {
				actions = append(actions, d.factories.NamespaceConfigAction.NewUpdate(&intent))
				updates++
			}
		} else {
			actions = append(actions, d.factories.NamespaceConfigAction.NewCreate(&intent))
			creates++
		}
	}
	for name := range current.NamespaceConfigs {
		if _, found := desired.NamespaceConfigs[name]; !found {
			actions = append(actions, d.factories.NamespaceConfigAction.NewDelete(name, desired))
			deletes++
		}
	}

	glog.Infof("NamespaceConfig operations: create %d, update %d, delete %d", creates, updates, deletes)
	return actions
}

func (d *Differ) clusterConfigActions(current, desired namespaceconfig.AllConfigs) []action.Interface {
	var actions []action.Interface
	actions = append(actions, d.nonCRDClusterConfigActions(current.ClusterConfig, desired.ClusterConfig)...)
	actions = append(actions, d.nonCRDClusterConfigActions(current.CRDClusterConfig, desired.CRDClusterConfig)...)

	return actions
}

func (d *Differ) nonCRDClusterConfigActions(current, desired *v1.ClusterConfig) []action.Interface {
	var actions []action.Interface
	if current == nil && desired == nil {
		return actions
	}
	if current == nil {
		actions = []action.Interface{d.factories.ClusterConfigAction.NewCreate(desired)}
	} else if desired == nil {
		actions = []action.Interface{d.factories.ClusterConfigAction.NewDelete(current.Name)}
	} else if !d.factories.ClusterConfigAction.Equal(desired, current) {
		actions = []action.Interface{d.factories.ClusterConfigAction.NewUpdate(desired)}
	}
	return actions
}

func (d *Differ) syncActions(current, desired namespaceconfig.AllConfigs) []action.Interface {
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

// SyncsInFirstOnly returns a list of sync names that are present in first, but
// not in second sync map.
func (d *Differ) SyncsInFirstOnly(first, second map[string]v1.Sync) []string {
	var inFirstOnly []string
	for _, sync := range first {
		if _, exists := second[sync.Name]; !exists {
			inFirstOnly = append(inFirstOnly, sync.Name)
		}
	}
	glog.V(4).Infof("Sync operations: %d deletes", len(inFirstOnly))
	if glog.V(8) {
		glog.Infof("first: %v, second: %v\ninFirstOnly: %v", first, second, inFirstOnly)
	}
	return inFirstOnly
}
