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

// Package validator is used for validating that the hierarchy specified using PolicyNodes forms
// a tree structure.
package validator

import (
	"github.com/davecgh/go-spew/spew"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/util/meta"
	"github.com/looplab/tarjan"
	"github.com/pkg/errors"
)

// Validator checks that a set of PolicyNode objects conform to certain constraints on the hierarchy.
type Validator struct {
	policyNodes map[string]v1.PolicyNode // name -> node
	// AllowMultipleRoots disables checks for multiple root nodes in the PolicyNode hierarchy, when true.
	AllowMultipleRoots bool
	// AllowOrphanAdds disables checks for adding PolicyNodes with non-existent parents.
	AllowOrphanAdds bool
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{
		policyNodes: map[string]v1.PolicyNode{},
	}
}

// FromMap creates a new validator populating the PolicyNodes from a map.
func FromMap(policyNodes map[string]v1.PolicyNode) *Validator {
	validator := New()
	for name, node := range policyNodes {
		validator.policyNodes[name] = node
	}
	return validator
}

// From creates a new Validator populating the nodes from a list of args.
func From(policyNodes ...*v1.PolicyNode) *Validator {
	validator := New()
	for _, policyNode := range policyNodes {
		validator.policyNodes[policyNode.Name] = *policyNode
	}
	return validator
}

// Add adds a node in the hierarchy to the validator.
func (s *Validator) Add(policyNode *v1.PolicyNode) error {
	nodeName := policyNode.Name
	if _, ok := s.policyNodes[nodeName]; ok {
		return errors.Errorf("PolicyNode %q already exists", nodeName)
	}
	if nodeName == "" {
		return errors.Errorf("PolicyNode does not have a name")
	}

	s.policyNodes[nodeName] = *policyNode
	return nil
}

// Update updates a node in the validator
func (s *Validator) Update(policyNode *v1.PolicyNode) error {
	nodeName := policyNode.Name
	if _, ok := s.policyNodes[nodeName]; !ok {
		return errors.Errorf("PolicyNode %q does not exist for update", nodeName)
	}

	s.policyNodes[nodeName] = *policyNode
	return nil
}

// Remove removes a node from the validator
func (s *Validator) Remove(policyNode *v1.PolicyNode) error {
	nodeName := policyNode.Name
	if _, ok := s.policyNodes[nodeName]; !ok {
		return errors.Errorf("PolicyNode %q does not exist for removal", nodeName)
	}

	for _, policyNode := range s.policyNodes {
		parent := policyNode.Spec.Parent
		if parent == nodeName {
			return errors.Errorf("PolicyNode %q is a parent and cannot be removed", nodeName)
		}
	}

	delete(s.policyNodes, nodeName)
	return nil
}

// Validate will validate that the tree structure satisfies the following:
// there is only one root (or at least one, depending on config)
// there are no cycles present
// Each leaf (non org node leaf) is a designated as a working namespace.
func (s *Validator) Validate() error {
	for _, checkFunction := range []func() error{
		s.checkRoots,
		s.checkCycles,
		s.checkPolicySpaceRoles,
		s.checkParents,
		s.checkResourceMetadata,
	} {
		if err := checkFunction(); err != nil {
			return err
		}
	}
	return nil
}

// checkRoots validates that there is only one (or at least one, if configured) root by checking
// that children of empty string (no parent) is the appropriate size.
func (s *Validator) checkRoots() error {
	if len(s.policyNodes) == 0 {
		// There are no PolicyNodes, so no reason to check for root issues.
		return nil
	}
	var roots []string
	for nodeName, node := range s.policyNodes {
		if node.Spec.Parent == v1.NoParentNamespace {
			roots = append(roots, nodeName)
		}
	}

	if len(roots) == 0 {
		return errors.New("at least one root (organization) node is required, none exist")
	}

	if len(roots) > 1 && !s.AllowMultipleRoots {
		return errors.Errorf(
			"exactly one root (organization) node is required, found %d: %v", len(roots), roots)
	}
	for _, nodeName := range roots {
		node := s.policyNodes[nodeName]
		if node.Spec.Type.IsNamespace() {
			return errors.Errorf("root node %q should not be a %s", nodeName, node.Spec.Type)
		}
	}
	return nil
}

// checkCycles checks that there are no cycles in the hierarchy
func (s *Validator) checkCycles() error {
	graph := map[interface{}][]interface{}{}

	for nodeName, node := range s.policyNodes {
		graph[nodeName] = []interface{}{node.Spec.Parent}
	}

	var cycles [][]interface{}
	output := tarjan.Connections(graph)
	for _, item := range output {
		if 2 <= len(item) {
			cycles = append(cycles, item)
		}
	}
	if len(cycles) != 0 {
		return errors.Errorf("found cycles %s, graph %s", cycles, spew.Sdump(graph))
	}
	return nil
}

// checkPolicySpaceRoles checks that there are no PolicySpaces that have Roles.
func (s *Validator) checkPolicySpaceRoles() error {
	for nodeName, node := range s.policyNodes {
		if node.Spec.Type.IsPolicyspace() && len(node.Spec.RolesV1) > 0 {
			return errors.Errorf(
				"PolicyNode %q designated as a policy space, but has Roles", nodeName)
		}
	}
	return nil
}

// checkParents checks that all the PolicyNodes have a parent (besides the root node).
func (s *Validator) checkParents() error {
	if s.AllowOrphanAdds {
		return nil
	}

	for nodeName, node := range s.policyNodes {
		parent := node.Spec.Parent
		_, ok := s.policyNodes[parent]
		if parent != v1.NoParentNamespace && !ok {
			return errors.Errorf("PolicyNode %q has no parent and is not a root node", nodeName)
		}
	}
	return nil
}

// checkMetadata checks that the resource's metadata is allowed by nomos.
func (s *Validator) checkResourceMetadata() error {
	validator := meta.NewValidator()
	for _, node := range s.policyNodes {
		if err := validator.Validate(node.Spec.RoleBindingsV1); err != nil {
			return err
		}
		if err := validator.Validate(node.Spec.RolesV1); err != nil {
			return err
		}
	}
	return nil
}
