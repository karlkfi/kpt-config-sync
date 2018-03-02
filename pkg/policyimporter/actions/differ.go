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

package actions

import (
	"sort"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	listers_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/policyhierarchy/v1"
	"github.com/google/stolos/pkg/syncer/actions"
)

type Differ struct {
	// Lister and interface needed to generate PolicyNode actions
	policyNodeLister    listers_v1.PolicyNodeLister
	policyNodeInterface typed_v1.PolicyNodeInterface
	// Lister and interface needed to generate ClusterPolicy actions
	clusterPolicyLister    listers_v1.ClusterPolicyLister
	clusterPolicyInterface typed_v1.ClusterPolicyInterface
	current, desired       v1.AllPolicies
}

// Differ will generate an ordered list of actions needed to transition policy from the current to
// desired state.
//
// Maintains the invariant that the policy node tree is valid (i.e. no cycles, parents pointing to existing nodes),
// assuming the current and desired state are valid themselves.
//
// More details about the algorithm can be found at docs/update-preserving-invariants.md
func NewDiffer(
	policyNodeLister listers_v1.PolicyNodeLister,
	policyNodeInterface typed_v1.PolicyNodeInterface,
	clusterPolicyLister listers_v1.ClusterPolicyLister,
	clusterPolicyInterface typed_v1.ClusterPolicyInterface) *Differ {
	return &Differ{
		policyNodeLister:       policyNodeLister,
		policyNodeInterface:    policyNodeInterface,
		clusterPolicyLister:    clusterPolicyLister,
		clusterPolicyInterface: clusterPolicyInterface,
	}
}

// Diff returns a list of actions that when applied, transitions the current state to desired state.
// Note that the invariants are only maintained if the actions are processed by a single thread in order.
// TODO(frankfarzan): Support ClusterPolicy.
func (d *Differ) Diff(current, desired v1.AllPolicies) []actions.Interface {
	d.current = current
	d.desired = desired

	creates := d.creates()
	updates := d.updates()
	deletes := d.deletes()
	glog.Infof("PolicyNode operations: create %d, update %d, delete %d", len(creates), len(updates), len(deletes))

	desiredByDepth := nodesByDepth(d.desired.PolicyNodes)
	currentByDepth := nodesByDepth(d.current.PolicyNodes)

	// Sort creates and updates by depth
	sort.Slice(creates, func(i, j int) bool {
		return desiredByDepth[creates[i]] < desiredByDepth[creates[j]]
	})
	sort.Slice(updates, func(i, j int) bool {
		return desiredByDepth[updates[i]] < desiredByDepth[updates[j]]
	})
	// Sort deletes by reverse depth in current tree
	sort.Slice(deletes, func(i, j int) bool {
		return currentByDepth[deletes[i]] > currentByDepth[deletes[j]]
	})

	var actions []actions.Interface
	for _, name := range append(creates, updates...) {
		node := d.desired.PolicyNodes[name]
		actions = append(actions, NewPolicyNodeUpsertAction(
			&node, d.policyNodeLister, d.policyNodeInterface))
	}
	for _, name := range deletes {
		node := d.current.PolicyNodes[name]
		actions = append(actions, NewPolicyNodeDeleteAction(
			&node, d.policyNodeLister, d.policyNodeInterface))
	}

	return actions
}

func (d *Differ) creates() []string {
	var creates []string

	for key, _ := range d.desired.PolicyNodes {
		if _, exists := d.current.PolicyNodes[key]; !exists {
			creates = append(creates, key)
		}
	}
	return creates
}

func (d *Differ) updates() []string {
	var updates []string

	for key, newnode := range d.desired.PolicyNodes {
		if oldnode, exists := d.current.PolicyNodes[key]; exists {
			if !equal(&newnode, &oldnode) {
				updates = append(updates, key)
			}
		}
	}
	return updates
}

func (d *Differ) deletes() []string {
	var deletes []string

	for key, _ := range d.current.PolicyNodes {
		if _, exists := d.desired.PolicyNodes[key]; !exists {
			deletes = append(deletes, key)
		}
	}
	return deletes
}

// Returns map of nodes to their depth, with root being at depth 0.
func nodesByDepth(nodes map[string]v1.PolicyNode) map[string]int {
	// Create a tree of the nodes, by mapping each node to a list of its children
	childrenByParent := map[string][]string{}

	for _, node := range nodes {
		if children, exists := childrenByParent[node.Spec.Parent]; exists {
			childrenByParent[node.Spec.Parent] = append(children, node.Name)
		} else {
			childrenByParent[node.Spec.Parent] = []string{node.Name}
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
