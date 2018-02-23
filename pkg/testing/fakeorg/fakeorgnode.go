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

package fakeorg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/util/policynode"
	"github.com/pkg/errors"
)

// Node wraps PolicyNode for fakeorg so we can track parent / child
type Node struct {
	policyNode *policyhierarchy_v1.PolicyNode
	parent     *Node
	children   map[string]*Node
}

// NewNode creates a new fake org node.
func NewNode(name string) *Node {
	return &Node{
		policyNode: policynode.NewPolicyNode(
			name,
			&policyhierarchy_v1.PolicyNodeSpec{}),
		children: map[string]*Node{},
	}
}

func (s *Node) addChild(child *Node) {
	child.policyNode.Spec.Parent = s.Name()
	child.parent = s
	s.children[child.Name()] = child
}

func (s *Node) removeChild(child *Node) {
	child.policyNode.Spec.Parent = ""
	child.parent = nil
	delete(s.children, child.Name())
}

func (s *Node) unparent() {
	parent := s.parent
	s.policyNode.Spec.Parent = ""
	s.parent = nil
	delete(parent.children, s.Name())
}

// Name returns the name of the underlying policy node
func (s *Node) Name() string {
	return s.policyNode.Name
}

// Parent reutrns the parent of the underlying policy node
func (s *Node) Parent() string {
	return s.policyNode.Spec.Parent
}

// PolicyNode returns the underlying policy node
func (s *Node) PolicyNode() *policyhierarchy_v1.PolicyNode {
	return s.policyNode
}

// IsLeaf returns true if the node is a leaf
func (s *Node) IsLeaf() bool {
	return len(s.children) == 0
}

// IsRoot returns true if the node is the root node.
func (s *Node) IsRoot() bool {
	return s.policyNode.Spec.Parent == ""
}

// Subtree returns a list of all descendants of the node.
func (s *Node) Subtree() []string {
	ret := []string{s.Name()}
	for _, childNode := range s.children {
		ret = append(ret, childNode.Subtree()...)
	}
	return ret
}

// IsDescendant returns true of other is a descendand of the current node.
func (s *Node) IsDescendant(other *Node) bool {
	for other != nil {
		if other.Parent() == s.Name() {
			return true
		}
		other = other.parent
	}
	return false
}

// Leaves returns a list of all leaves of the subtree rooted at the node.
// Note that if the current node is a leaf, it will be returned as the only leaf.
func (s *Node) Leaves() []string {
	ret := []string{}
	if len(s.children) == 0 {
		return []string{s.policyNode.Name}
	}
	for _, childNode := range s.children {
		ret = append(ret, childNode.Leaves()...)
	}
	return ret
}

// Write writes the policy node to a file in the directory with the same name as the namespace
func (s *Node) Write(outputDir string) error {
	filePath := filepath.Join(outputDir, fmt.Sprintf("%s.json", s.Name()))
	return s.WriteToFile(filePath)
}

// WriteToFile writes the policy node to a file at path
func (s *Node) WriteToFile(filePath string) error {
	glog.Infof("Writing to %s", filePath)
	nodeBytes, err := json.MarshalIndent(s.PolicyNode(), "", "  ")
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal PolicyNode %s", spew.Sdump(s.policyNode))
	}
	err = ioutil.WriteFile(filePath, nodeBytes, 0644)
	if err != nil {
		return errors.Wrapf(err, "Failed write file to %s", filePath)
	}
	return nil
}
