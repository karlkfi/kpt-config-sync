// Reviewed by sunilarora
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
	"sort"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/policyimporter"
)

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
func (d *Differ) Diff(current, desired v1.AllPolicies) []action.Interface {
	var actions []action.Interface
	actions = append(actions, d.policyNodeActions(current, desired)...)
	actions = append(actions, d.clusterPolicyActions(current, desired)...)
	actions = append(actions, d.syncUpserts(current, desired)...)
	return actions
}

func (d *Differ) policyNodeActions(current, desired v1.AllPolicies) []action.Interface {
	creates := d.nodeCreates(current, desired)
	updates := d.nodeUpdates(current, desired)
	deletes := d.nodeDeletes(current, desired)
	glog.Infof("PolicyNode operations: create %d, update %d, delete %d", len(creates), len(updates), len(deletes))
	policyimporter.Metrics.Operations.WithLabelValues("create").Add(float64(len(creates)))
	policyimporter.Metrics.Operations.WithLabelValues("update").Add(float64(len(updates)))
	policyimporter.Metrics.Operations.WithLabelValues("delete").Add(float64(len(deletes)))

	desiredByDepth := d.nodesByDepth(desired.PolicyNodes)
	currentByDepth := d.nodesByDepth(current.PolicyNodes)

	// Sort nodeCreates and nodeUpdates by depth
	sort.Slice(creates, func(i, j int) bool {
		return desiredByDepth[creates[i]] < desiredByDepth[creates[j]]
	})
	sort.Slice(updates, func(i, j int) bool {
		return desiredByDepth[updates[i]] < desiredByDepth[updates[j]]
	})
	// Sort nodeDeletes by reverse depth in current tree
	sort.Slice(deletes, func(i, j int) bool {
		return currentByDepth[deletes[i]] > currentByDepth[deletes[j]]
	})

	var actions []action.Interface
	for _, name := range creates {
		node := desired.PolicyNodes[name]
		actions = append(actions, d.factories.PolicyNodeAction.NewCreate(&node))
	}
	for _, name := range updates {
		node := desired.PolicyNodes[name]
		actions = append(actions, d.factories.PolicyNodeAction.NewUpdate(&node))
	}
	for _, name := range deletes {
		node := current.PolicyNodes[name]
		actions = append(actions, d.factories.PolicyNodeAction.NewDelete(node.Name))
	}
	return actions
}

func (d *Differ) clusterPolicyActions(current, desired v1.AllPolicies) []action.Interface {
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

func (d *Differ) syncUpserts(current, desired v1.AllPolicies) []action.Interface {
	var actions []action.Interface
	var creates, updates int
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
	glog.Infof("Sync operations: %d updates, %d creates", updates, creates)

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

// SyncReductions returns a list of Syncs to be written during the deletion phase (meaning before
// PolicyNode and ClusterPolicy updates).
//
// The returned list includes Syncs that are present in both current and desired state, but which
// lose some GVKs when going from current to desired. And it returns them without those lost GVKs.
// In other words, these Syncs are an intersection of the current and desired state.
//
// Any Syncs which do not lose GVKs are not included.
//
// Example:
// current:
//   v1.AllPolicies{
//     Syncs: []v1alpha1.Sync{{
//         Name: "ResourceQuota",
//         Kinds: []v1alpha1.SyncKind{{
//             Kind: "ResourceQuota",
//             Versions: []v1alpha1.SyncVersion{
//               {Version: "v1"},
//               {Version: "v2"},
//             },
//         }},
//     }},
//   }
// desired:
//   v1.AllPolicies{
//     Syncs: []v1alpha1.Sync{{
//         Name: "ResourceQuota",
//         Kinds: []v1alpha1.SyncKind{{
//             Kind: "ResourceQuota",
//             Versions: []v1alpha1.SyncVersion{
//               {Version: "v1"},
//               {Version: "v3"},
//             },
//         }},
//     }},
//   }
//
// returned:
//   []v1.AllPolicies{{
//       Syncs: []v1alpha1.Sync{{
//           Name: "ResourceQuota",
//           Kinds: []v1alpha1.SyncKind{{
//               Kind: "ResourceQuota",
//               Versions: []v1alpha1.SyncVersion{
//                 {Version: "v1"},
//               },
//           }},
//       }},
//   }}
//
// This method is required so that Syncer can finish removing all Watchers before it starts applying
// PolicyNode updates.
func (d *Differ) SyncReductions(current, desired map[string]v1alpha1.Sync) []v1alpha1.Sync {
	var toReduce []v1alpha1.Sync
	for _, sync := range current {
		if des, exists := desired[sync.Name]; exists {
			if intersection := d.reduce(sync, des); intersection != nil {
				toReduce = append(toReduce, *intersection)
			}
		}
	}
	glog.Infof("Sync operations: %d reductions", len(toReduce))
	return toReduce
}

// reduce calculates and returns the intersection between current and desired. It only returns the
// result if it is not equal to current. Returns nil otherwise.
func (d *Differ) reduce(current, desired v1alpha1.Sync) *v1alpha1.Sync {
	if d.factories.SyncAction.Equal(&current, &desired) {
		return nil
	}
	desiredSyncs := make(map[string]map[string]map[string]struct{})
	for _, g := range desired.Spec.Groups {
		group, ge := desiredSyncs[g.Group]
		if !ge {
			group = make(map[string]map[string]struct{})
			desiredSyncs[g.Group] = group
		}
		for _, k := range g.Kinds {
			kind, ke := group[k.Kind]
			if !ke {
				kind = make(map[string]struct{})
				group[k.Kind] = kind
			}
			for _, v := range k.Versions {
				if _, ve := kind[v.Version]; !ve {
					kind[v.Version] = struct{}{}
				}
			}
		}
	}

	var diffExists bool
	i := v1alpha1.Sync{TypeMeta: current.TypeMeta, ObjectMeta: current.ObjectMeta}
	for _, oldG := range current.Spec.Groups {
		gm, ge := desiredSyncs[oldG.Group]
		if !ge {
			diffExists = true
			continue
		}
		newG := v1alpha1.SyncGroup{Group: oldG.Group}
		for _, oldK := range oldG.Kinds {
			km, ke := gm[oldK.Kind]
			if !ke {
				diffExists = true
				continue
			}
			newK := v1alpha1.SyncKind{Kind: oldK.Kind}
			for _, oldV := range oldK.Versions {
				if _, ve := km[oldV.Version]; !ve {
					diffExists = true
					continue
				}
				newK.Versions = append(newK.Versions, oldV)
			}
			newG.Kinds = append(newG.Kinds, newK)
		}
		i.Spec.Groups = append(i.Spec.Groups, newG)
	}
	if diffExists {
		return &i
	}
	// Even though we check for equality at the beginning of the method, this is reachable. That's
	// because two Syncs may not be Equal() but nonetheless have the same set of GVKs (in that case,
	// they may have different CompareFields, or the ordering of GVKs could be different.) Or, desired
	// could be a superset of current.
	return nil
}

func (d *Differ) nodeCreates(current, desired v1.AllPolicies) []string {
	var creates []string

	for _, nodeName := range d.nodeNames(desired.PolicyNodes) {
		if _, exists := current.PolicyNodes[nodeName]; !exists {
			creates = append(creates, nodeName)
		}
	}
	return creates
}

func (d *Differ) nodeUpdates(current, desired v1.AllPolicies) []string {
	var updates []string

	for _, nodeName := range d.nodeNames(desired.PolicyNodes) {
		newNode := desired.PolicyNodes[nodeName]
		if oldNode, exists := current.PolicyNodes[nodeName]; exists {
			if !d.factories.PolicyNodeAction.Equal(&newNode, &oldNode) {
				updates = append(updates, nodeName)
			}
		}
	}
	return updates
}

func (d *Differ) nodeDeletes(current, desired v1.AllPolicies) []string {
	var deletes []string

	for _, nodeName := range d.nodeNames(current.PolicyNodes) {
		if _, exists := desired.PolicyNodes[nodeName]; !exists {
			deletes = append(deletes, nodeName)
		}
	}
	return deletes
}

// Returns map of nodes to their depth, with root being at depth 0.
func (d *Differ) nodesByDepth(nodes map[string]v1.PolicyNode) map[string]int {
	// Create a tree of the nodes, by mapping each node to a list of its children
	childrenByParent := map[string][]string{}

	for _, nodeName := range d.nodeNames(nodes) {
		node := nodes[nodeName]
		if children, exists := childrenByParent[node.Spec.Parent]; exists {
			childrenByParent[node.Spec.Parent] = append(children, nodeName)
		} else {
			childrenByParent[node.Spec.Parent] = []string{nodeName}
		}
	}

	// Traverse the tree by starting from the root (child of NoParentNamespace)
	// and assigning depth to each layer of children. (Basically BFS)
	var nodesAtNextDepth []string
	nodesByDepth := map[string]int{}
	nodesAtDepth := childrenByParent[v1.NoParentNamespace]
	depth := 0

	for len(nodesAtDepth) != 0 {
		for _, node := range nodesAtDepth {
			nodesByDepth[node] = depth
			nodesAtNextDepth = append(nodesAtNextDepth, childrenByParent[node]...)
		}

		depth++
		nodesAtDepth = nodesAtNextDepth
		nodesAtNextDepth = []string{}
	}

	return nodesByDepth
}

// nodeNames returns the policy node names from the map. When the results of a
// diff should be deterministic, we sort the nodeNames returned in order to
// bypass randomness associated with iterating over maps.
func (d *Differ) nodeNames(nodes map[string]v1.PolicyNode) []string {
	var nodeNames []string
	for nodeName := range nodes {
		nodeNames = append(nodeNames, nodeName)
	}
	if !d.SortDiff {
		return nodeNames
	}
	sort.Strings(nodeNames)
	return nodeNames
}
