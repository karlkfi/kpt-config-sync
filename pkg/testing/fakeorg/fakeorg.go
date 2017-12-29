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

// Package fakeorg generates a fake organization hierarchy and then performs mutations on it.
package fakeorg

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

// FakeOrg is a fake org hierarchy
type FakeOrg struct {
	orgNode *Node            // The root node
	nodes   map[string]*Node // All nodes including root node
}

// New creates a new fake org with just the org node populated.
func New(orgName string) *FakeOrg {
	orgNode := NewNode(orgName)
	return &FakeOrg{
		nodes:   map[string]*Node{orgName: orgNode},
		orgNode: orgNode,
	}
}

// Validate checks the internal structure of FakeOrg
func (s *FakeOrg) Validate() {
	// Check parent/child node consistency, all nodes except for org node must have a parent,
	// all parent nodes must have node as child.
	for _, node := range s.nodes {
		if node == s.orgNode {
			if node.policyNode.Spec.Parent != "" {
				panic(errors.Errorf("org node should not have parent name"))
			}
			if node.parent != nil {
				panic(errors.Errorf("org node should not have parent node"))
			}
		} else {
			if node.Parent() == "" {
				panic(errors.Errorf("node does not have parent name"))
			}
			parentNode, found := s.nodes[node.Parent()]
			if !found {
				panic(errors.Errorf("node parent name does not exist as node"))
			}
			if node.parent != parentNode {
				panic(errors.Errorf("node does not have parent node"))
			}

			parentChild, found := parentNode.children[node.Name()]
			if parentChild == nil {
				panic(errors.Errorf("child node is not in parent"))
			}
			if parentChild != node {
				panic(errors.Errorf("child node does not match child in parent"))
			}

			// Validate children of current node have correct parent pointer / name
			for _, childNode := range node.children {
				if childNode.Parent() != node.Name() {
					panic(errors.Errorf("Child of parent does not have correct parent node name"))
				}
				if childNode.parent != node {
					panic(errors.Errorf("Child of parent does not have correct parent node pointer"))
				}
			}
		}
	}

	// Cycle check, mark descendants of root. Since we have already checked the parent / child
	// pointers for descendants of the root node, it's not possible to have a cycle descending from
	// the org node so we mark visited nodes and if we have leftovers then there is a cycle.
	visited := map[string]bool{}
	for _, nodeName := range s.orgNode.Subtree() {
		visited[nodeName] = true
	}
	for name := range s.nodes {
		if !visited[name] {
			panic(errors.Errorf(
				"Node %s not visited in descendants of org node\n%s", name, spew.Sdump(s)))
		}
	}
}

// RootNode returns the root node for the fake org
func (s *FakeOrg) RootNode() *Node {
	return s.orgNode
}

// Len returns the number of nodes in the fake org
func (s *FakeOrg) Len() int {
	return len(s.nodes)
}

// NodeNames returns the name of all ndoes in the fake org
func (s *FakeOrg) NodeNames() []string {
	ret := []string{}
	for name := range s.nodes {
		ret = append(ret, name)
	}
	return ret
}

// GetNode returns a node by name from the fake node, will panic if node is not found.
func (s *FakeOrg) GetNode(name string) *Node {
	s.assertContains(true, "GetNode", name)
	return s.nodes[name]
}

// Nodes returns all nodes from the fake org
func (s *FakeOrg) Nodes() []*Node {
	ret := []*Node{}
	for _, node := range s.nodes {
		ret = append(ret, node)
	}
	return ret
}

// Contains returns true if the fakeOrg contains a node with the given name.
func (s *FakeOrg) Contains(namespace string) bool {
	return s.nodes[namespace] != nil
}

// assertContains will panic if the return value of Contains does not match the assertContains arg.
func (s *FakeOrg) assertContains(assertContains bool, message string, name string) {
	if s.Contains(name) != assertContains {
		complaint := map[bool]string{true: "does not contain", false: "already contains"}[assertContains]
		panic(errors.Errorf("%s: FakeOrg %s node %s:\n%s", message, complaint, name, spew.Sdump(s)))
	}
}

// assertContainsNode will panic if the result of Contains does not match the assertContains arg
// for the name of the node.
func (s *FakeOrg) assertContainsNode(assertContains bool, message string, node *Node) {
	s.assertContains(assertContains, message, node.Name())
}

// AddNode adds a node to the fake org as the child of parent
func (s *FakeOrg) AddNode(parent *Node, child *Node) {
	s.assertContainsNode(false, "Added invalid child node", child)
	s.assertContainsNode(true, "Added to invalid parent node", parent)

	s.nodes[child.Name()] = child
	parent.addChild(child)
}

// RemoveNode removes a node from the org. Note that removing the org node is not allowed, and
// the node removed must be a leaf node.
func (s *FakeOrg) RemoveNode(node *Node) {
	s.assertContainsNode(true, "Invalid remove", node)
	if node == s.orgNode {
		panic("Cannot remove orgnode")
	}

	if !node.IsLeaf() {
		panic(errors.Errorf("Cannot remove non-leaf node"))
	}

	node.unparent()
	delete(s.nodes, node.Name())
}

// ReparentNode will change the parent of child to newParent.  This will fail if it introduces a
// cycle into the fakeorg.
func (s *FakeOrg) ReparentNode(newParent *Node, child *Node) {
	s.assertContainsNode(true, "Reparent invalid child node", child)
	s.assertContainsNode(true, "Reparent invalid parent node", newParent)

	if child.IsDescendant(newParent) {
		panic(errors.Errorf("Cannot set parent to descendant of node."))
	}

	child.unparent()
	newParent.addChild(child)
}
