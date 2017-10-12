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

// Package validator is used for validating that the heirarchy specified using policy nodes forms
// a tree structure.
package validator

import (
	"strings"

	"github.com/davecgh/go-spew/spew"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/looplab/tarjan"
	"github.com/pkg/errors"
)

type Validator struct {
	policyNodes map[string]*policyhierarchy_v1.PolicyNode // name -> node
	children    map[string][]string                       // parent -> []children
	parents     map[string]string                         // child -> parent
}

func New() *Validator {
	return &Validator{
		policyNodes: map[string]*policyhierarchy_v1.PolicyNode{},
		children:    map[string][]string{},
		parents:     map[string]string{},
	}
}

// Add adds a node in the hierarchy to the validator.
func (s *Validator) Add(policyNode *policyhierarchy_v1.PolicyNode) error {
	nodeName := policyNode.Name
	if s.policyNodes[nodeName] != nil {
		return errors.Errorf("Policy node %s already exists!", nodeName)
	}
	if nodeName == "" {
		return errors.Errorf("Policy node does not have a name")
	}

	s.policyNodes[nodeName] = policyNode
	parentName := policyNode.Spec.Parent
	s.parents[nodeName] = parentName
	s.children[parentName] = append(s.children[parentName], nodeName)
	return nil
}

// Validate will validate that the tree structure satisfies the following:
// there is only one root
// there are no cycles present
// Each leaf (non org node leaf) is a designated as a working namespace.
func (s *Validator) Validate() error {
	for _, checkFunction := range []func() error{
		s.checkMultipleRoots, s.checkCycles, s.checkWorkingNamespace} {
		if err := checkFunction(); err != nil {
			return err
		}
	}
	return nil
}

// checkMultipleRoots validates that there is only one root by checking that children of empty string
// (no parent) is of size 1
func (s *Validator) checkMultipleRoots() error {
	rootNodeList := s.children[""]
	if len(rootNodeList) != 1 {
		return errors.Errorf(
			"Exactly one root (organization) node is required, found %d", len(rootNodeList))
	}
	return nil
}

// checkWorkingNamespace checks that all non-root leaves are working namespaces while internal nodes
// are not working namespaces
func (s *Validator) checkWorkingNamespace() error {
	for nodeName, node := range s.policyNodes {
		if node.Spec.Parent == "" {
			// Root node should not be a working namespace
			if node.Spec.WorkingNamespace {
				return errors.Errorf("Root node %s should not be a working namespace", nodeName)
			}
			continue
		}

		children := s.children[nodeName]
		if node.Spec.WorkingNamespace && len(children) != 0 {
			return errors.Errorf(
				"Node %s designated as working namespace, but has children %s", nodeName, strings.Join(children, ", "))
		}
		if !node.Spec.WorkingNamespace && len(children) == 0 {
			return errors.Errorf(
				"Node %s not designated as working namespace, but has no children", nodeName)
		}
	}
	return nil
}

// checkCycles checks that there are no cycles in the hierarchy
func (s *Validator) checkCycles() error {
	graph := map[interface{}][]interface{}{}

	for child, parent := range s.parents {
		graph[child] = []interface{}{parent}
	}

	cycles := [][]interface{}{}
	output := tarjan.Connections(graph)
	for _, item := range output {
		if 2 <= len(item) {
			cycles = append(cycles, item)
		}
	}
	if len(cycles) != 0 {
		return errors.Errorf("Found cycles %s, graph %s", cycles, spew.Sdump(graph))
	}

	return nil
}
