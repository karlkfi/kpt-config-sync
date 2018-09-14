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

package hierarchy

import (
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/nomos/v1"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestNodeSpec is a convenience struct for generating a hierarchy of PolicyNode objects.
type TestNodeSpec struct {
	Name   string
	Parent string
}

// TestNodeSpecs is a tiny type for a list of TestNodeSpec which represents a full hierarchy.
type TestNodeSpecs []TestNodeSpec

// Ancestry is an alternate implementation of ancestry for testing.
func (s TestNodeSpecs) Ancestry(name string) []string {
	if name == "" {
		return []string{}
	}

	for _, spec := range s {
		if spec.Name == name {
			ancestors := s.Ancestry(spec.Parent)
			return append(ancestors, name)
		}
	}
	panic(errors.Errorf("Could not find node %s", name))
}

// Subtree is an alternate implementation of subtree for testing.
func (s TestNodeSpecs) Subtree(name string) []string {
	nodes := []string{name}
	for _, spec := range s {
		if spec.Parent == name {
			nodes = append(nodes, s.Subtree(spec.Name)...)
		}
	}
	return nodes
}

// TestAggregatedOutput is a fake output type for our TestAggregatedNode.
type TestAggregatedOutput struct {
	meta_v1.TypeMeta
	meta_v1.ObjectMeta

	Ancestry []string
}

// TestAggregatedNode is a simple implementor of AggregatedNode for test purposes.
type TestAggregatedNode struct {
	Ancestry []string
}

// Aggregated implements AggregatedNode
func (s *TestAggregatedNode) Aggregated(childNode *policyhierarchy_v1.PolicyNode) AggregatedNode {
	ancestry := make([]string, len(s.Ancestry))
	copy(ancestry, s.Ancestry)
	ancestry = append(ancestry, childNode.Name)
	return &TestAggregatedNode{Ancestry: ancestry}
}

// Generate implements AggregatedNode
func (s *TestAggregatedNode) Generate() Instances {
	return Instances{&TestAggregatedOutput{Ancestry: s.Ancestry}}
}

// NewTestAggregatedNode creates a new TestAggregatedNode for testing.
func NewTestAggregatedNode() AggregatedNode {
	return &TestAggregatedNode{}
}
