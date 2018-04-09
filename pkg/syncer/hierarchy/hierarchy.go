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

	"github.com/golang/glog"
	policyhierarchyinformer_v1 "github.com/google/nomos/clientgen/informers/externalversions/policyhierarchy/v1"
	policyhierarchylister_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
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

// Instances represents all nodes in a
type Instances []meta_v1.Object

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
	glog.Fatal("not implemented")
	return nil
}

// Node returns the node that was requested during the ancestry lookup.
func (s Ancestry) Node() *policyhierarchy_v1.PolicyNode {
	return s[0]
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
	glog.Fatal("not implemented.")
	return nil, nil
}

// Subtree returns the name of all nodes in the subtree rooted at the named policy node. Returns a
// NotFoundError if the node with the given name was not found. Since there are only parent pointers
// it's not possible to detect an incomplete hierarchy.
func (s *Hierarchy) Subtree(name string) ([]string, error) {
	glog.Fatal("not implemented.")
	return nil, nil
}
