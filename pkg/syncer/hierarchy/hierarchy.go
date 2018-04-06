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

// Package hierarchy provides support for aggregating hierarchical policies that are specified in
// PolicyNodes.
package hierarchy

import (
	"fmt"
	"sort"

	"strings"

	policyhierarchyinformer_v1 "github.com/google/nomos/clientgen/informers/externalversions/policyhierarchy/v1"
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

// IncompleteHierarchyError is returned when ancestry is not able to retrieve all ancestors to the
// root node.
type IncompleteHierarchyError struct {
	name string
}

// Error implements error
func (s *IncompleteHierarchyError) Error() string {
	return fmt.Sprintf("ancestor policy node %s missing from hierarchy", s.name)
}

// IsIncompleteHierarchyError returns true if the error is a NotFoundError
func IsIncompleteHierarchyError(err error) bool {
	_, ok := err.(*IncompleteHierarchyError)
	return ok
}

// Instances represents all nodes in an AggregatedNode.
type Instances []meta_v1.Object

func (m Instances) Len() int {
	return len(m)
}
func (m Instances) Less(i, j int) bool {
	return strings.Compare(m[i].GetName(), m[j].GetName()) < 0
}
func (m Instances) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}
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

	ancestry := Ancestry{node}
	current := node.Spec.Parent
	for current != "" {
		node, err = s.lister.Get(current)
		if err != nil {
			if api_errors.IsNotFound(err) {
				return nil, &IncompleteHierarchyError{current}
			}
			panic("Lister returned error other than not found, this should not happen")
		}

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
