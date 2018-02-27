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

	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	listers_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/policyhierarchy/v1"
	"github.com/google/stolos/pkg/syncer/actions"
)

type Generator struct {
	// Map of existing policy nodes by their names
	oldNodes map[string]v1.PolicyNode
	// Map of the newNodes state of policy nodes by their names
	newNodes map[string]v1.PolicyNode

	// Lister and interface needed to generate PolicyNode actions
	policyNodeLister    listers_v1.PolicyNodeLister
	policyNodeInterface typed_v1.PolicyNodeInterface
}

// Generator will generate an ordered list of actions needed to update
// policy nodes from an old/existing state to a new provided state,
// while always maintaining the invariant that the policy node tree is valid
// - no cycles, parents pointing to existing nodes.
// This assumes the oldNodes and newNodes state are valid trees themselves.
//
// More details about the algorithm can be found at docs/update-preserving-invariants.md
func NewGenerator(oldNodes, newNodes []v1.PolicyNode,
	policyNodeLister listers_v1.PolicyNodeLister,
	policyNodeInterface typed_v1.PolicyNodeInterface) *Generator {

	oldMap := map[string]v1.PolicyNode{}
	newMap := map[string]v1.PolicyNode{}

	for _, node := range newNodes {
		newMap[node.Name] = node
	}
	for _, node := range oldNodes {
		oldMap[node.Name] = node
	}

	return &Generator{
		oldNodes:            oldMap,
		newNodes:            newMap,
		policyNodeLister:    policyNodeLister,
		policyNodeInterface: policyNodeInterface,
	}
}

// This is the main method that will generate the actions as described above.
// Note that the invariants are only maintained if the actions are processed
// by a single thread in order.
func (g *Generator) GenerateActions() []actions.Interface {
	creates := g.creates()
	updates := g.updates()
	deletes := g.deletes()

	newNodesByDepth := nodesByDepth(g.newNodes)
	oldNodesByDepth := nodesByDepth(g.oldNodes)

	// Sort creates and updates by depth
	sort.Slice(creates, func(i, j int) bool {
		return newNodesByDepth[creates[i]] < newNodesByDepth[creates[j]]
	})
	sort.Slice(updates, func(i, j int) bool {
		return newNodesByDepth[updates[i]] < newNodesByDepth[updates[j]]
	})
	// Sort deletes by reverse depth in oldNodes tree
	sort.Slice(deletes, func(i, j int) bool {
		return oldNodesByDepth[deletes[i]] > oldNodesByDepth[deletes[j]]
	})

	var actions []actions.Interface
	for _, name := range append(creates, updates...) {
		node := g.newNodes[name]
		actions = append(actions, NewPolicyNodeUpsertAction(
			&node, g.policyNodeLister, g.policyNodeInterface))
	}
	for _, name := range deletes {
		node := g.oldNodes[name]
		actions = append(actions, NewPolicyNodeDeleteAction(
			&node, g.policyNodeLister, g.policyNodeInterface))
	}

	return actions
}

func (g *Generator) creates() []string {
	var creates []string

	for key, _ := range g.newNodes {
		if _, exists := g.oldNodes[key]; !exists {
			creates = append(creates, key)
		}
	}
	return creates
}

func (g *Generator) updates() []string {
	var updates []string

	for key, newnode := range g.newNodes {
		if oldnode, exists := g.oldNodes[key]; exists {
			if !equal(&newnode, &oldnode) {
				updates = append(updates, key)
			}
		}
	}
	return updates
}

func (g *Generator) deletes() []string {
	var deletes []string

	for key, _ := range g.oldNodes {
		if _, exists := g.newNodes[key]; !exists {
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
