/*
Copyright 2018 The Nomos Authors.
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
// Reviewed by sunilarora

// Package hierarchy provides support for aggregating hierarchical policies that are specified in
// PolicyNodes.
package hierarchy

import (
	"fmt"
	"sort"

	"strings"

	policyhierarchyinformer_v1 "github.com/google/nomos/clientgen/informers/policyhierarchy/policyhierarchy/v1"
	policyhierarchylister_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/parentindexer"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// NotFoundError is returned if a requested node is not found.
type NotFoundError struct {
	name string
}

// Error implements error
func (s *NotFoundError) Error() string {
	return fmt.Sprintf("policy node %s not found", s.name)
}

// IsNotFoundError returns true if the error is a NotFoundError
func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// ConsistencyError is returned when the ancestry is in an inconsistent state. This can occur if
// an update to the PolicyNode objects happens in an order where the hierarchy is slightly disrupted.
type ConsistencyError struct {
	errType  string
	ancestry Ancestry
	missing  string
}

// Error implements error
func (s *ConsistencyError) Error() string {
	var vals []string
	for _, item := range s.ancestry {
		vals = append(vals, fmt.Sprintf("[%s:%s]", item.Name, item.Spec.Type))
	}
	if len(s.missing) != 0 {
		vals = append(vals, fmt.Sprintf("[%s:%s]", s.missing, "NotFound"))
	}
	return fmt.Sprintf("inconsistent hierarchy (%s): %s", s.errType, strings.Join(vals, " -> "))
}

// IsConsistencyError returns true if the error is a ConsistencyError
func IsConsistencyError(err error) bool {
	_, ok := err.(*ConsistencyError)
	return ok
}

// Instances represents all nodes in an AggregatedNode.
type Instances []meta_v1.Object

// Len implements sort.Interface
func (m Instances) Len() int {
	return len(m)
}

// Less implements sort.Interface
func (m Instances) Less(i, j int) bool {
	return strings.Compare(m[i].GetName(), m[j].GetName()) < 0
}

// Swap implements sort.Interface
func (m Instances) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

// Sort sorts the list of objects by name in ascending order.
func (m Instances) Sort() {
	sort.Sort(m)
}

// AggregatedNode is the interface that specific policy types implement for merge operations.
type AggregatedNode interface {
	// Aggregated produces the result of aggregating the current node with a policy node that is a
	// child of the current node.
	Aggregated(node *policyhierarchy_v1.PolicyNode) AggregatedNode

	// Generate returns all policies that are specified by the aggregation operations.
	Generate() Instances
}

// AggregatedNodeFactory is a function that returns a new AggregatedNode
type AggregatedNodeFactory func() AggregatedNode

// Ancestry represents the ancestry of a given policy node where the 0'th element is the requested
// node and last node is the root of the hierarchy.
type Ancestry []*policyhierarchy_v1.PolicyNode

// Aggregate takes an AggregatedNodeFactory and produces instances based on a hierarchical
// evaluation.
func (s Ancestry) Aggregate(factory AggregatedNodeFactory) Instances {
	aggregate := factory()
	for i := len(s) - 1; i >= 0; i-- {
		aggregate = aggregate.Aggregated(s[i])
	}
	return aggregate.Generate()
}

// Node returns the node that was requested during the ancestry lookup.
func (s Ancestry) Node() *policyhierarchy_v1.PolicyNode {
	return s[0]
}

// String implements Stringer
func (s Ancestry) String() string {
	var names []string
	for _, policyNode := range s {
		names = append(names, policyNode.Name)
	}
	return strings.Join(names, " -> ")
}

// TokenMap returns a map of node name to current import token for all nodes in the ancestry.
func (s Ancestry) TokenMap() map[string]string {
	tokens := make(map[string]string, len(s))
	for _, node := range s {
		tokens[node.Name] = node.Spec.ImportToken
	}
	return tokens
}

// Interface is the interface that the Hierarchy object fulfills.
type Interface interface {
	Ancestry(name string) (Ancestry, error)
	Subtree(name string) ([]string, error)
}

// Hierarchy performs common operations involved in hierarchical evaluation.
type Hierarchy struct {
	lister   policyhierarchylister_v1.PolicyNodeLister
	informer cache.SharedIndexInformer
}

// Hierarchy implements HierarchyInterface
var _ Interface = &Hierarchy{}

// New returns new Hierarchy object.
func New(informer policyhierarchyinformer_v1.PolicyNodeInformer) *Hierarchy {
	return &Hierarchy{
		lister:   informer.Lister(),
		informer: informer.Informer(),
	}
}

// Ancestry returns the ancestry of the named policy node. Returns a NotFoundError if the resource
// was not found. For the event of an incomplete hierarchy, an IncompleteHierarchyError will be
// returned.
func (s *Hierarchy) Ancestry(name string) (Ancestry, error) {
	node, err := s.lister.Get(name)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil, &NotFoundError{name}
		}
		panic("Lister returned error other than not found, this should not happen")
	}

	names := map[string]bool{name: true}
	ancestry := Ancestry{node}
	current := node.Spec.Parent
	for current != "" {
		node, err = s.lister.Get(current)
		if err != nil {
			if api_errors.IsNotFound(err) {
				return nil, &ConsistencyError{
					errType: "not found", ancestry: ancestry, missing: current}
			}
			panic("Lister returned error other than not found, this should not happen")
		}

		if !node.Spec.Type.IsPolicyspace() {
			return nil, &ConsistencyError{errType: "invalid parent", ancestry: ancestry}
		}

		if names[node.Name] {
			ancestry = append(ancestry, node)
			return nil, &ConsistencyError{errType: "cycle", ancestry: ancestry}
		}

		names[node.Name] = true
		ancestry = append(ancestry, node)
		current = node.Spec.Parent
	}

	return ancestry, nil
}

// Subtree returns the name of all nodes in the subtree rooted at the named policy node. Returns a
// NotFoundError if the node with the given name was not found. Since there are only parent pointers
// it's not possible to detect an incomplete hierarchy.
func (s *Hierarchy) Subtree(name string) ([]string, error) {
	if _, err := s.lister.Get(name); err != nil {
		if api_errors.IsNotFound(err) {
			return nil, &NotFoundError{name}
		}
		panic("Lister returned error other than not found, this should not happen")
	}
	return s.subtree(name), nil
}

func (s *Hierarchy) subtree(name string) []string {
	subtree := []string{name}
	children, err := parentindexer.GetChildren(s.informer, name)
	if err != nil {
		return subtree
	}
	for _, child := range children {
		subtree = append(subtree, s.subtree(child)...)
	}
	return subtree
}
