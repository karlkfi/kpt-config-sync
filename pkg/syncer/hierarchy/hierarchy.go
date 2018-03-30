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
	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	policyhierarchyinformer_v1 "github.com/google/nomos/pkg/client/informers/externalversions/policyhierarchy/v1"
	policyhierarchylister_v1 "github.com/google/nomos/pkg/client/listers/policyhierarchy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// Instances represents all nodes in a
type Instances map[string]meta_v1.Object

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

// Ancestry represents the ancestry of a given policy node where the 0'th element is the root of the
// hierarchy and the last element is policy node itself.
type Ancestry []*policyhierarchy_v1.PolicyNode

// Aggregate takes an AggregatedNodeFactory and produces instances based on a hierarchical
// evaluation.
func (s Ancestry) Aggregate(factory AggregatedNodeFactory) Instances {
	glog.Fatal("not implemented")
	return nil
}

// Hierarchy performs common operations involved in hierarchical evaluation.
type Hierarchy struct {
	lister   policyhierarchylister_v1.PolicyNodeLister
	informer cache.SharedIndexInformer
}

// New returns new Hierarchy object.
func New(informer policyhierarchyinformer_v1.PolicyNodeInformer) *Hierarchy {
	return &Hierarchy{
		lister:   informer.Lister(),
		informer: informer.Informer(),
	}
}

// Ancestry returns the ancestry of the named policy node.
func (s *Hierarchy) Ancestry(name string) Ancestry {
	glog.Fatal("not implemented.")
	return nil
}

// Subtree returns the name of all nodes in the subtree rooted at the named policy node.
func (s *Hierarchy) Subtree(name string) []string {
	glog.Fatal("not implemented.")
	return nil
}
