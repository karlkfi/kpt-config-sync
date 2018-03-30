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

package hierarchy

import (
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Convenience for generating hierarchy.
type TestNodeSpec struct {
	Name   string
	Parent string
}

func createTestNode(name, parent string) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Parent: parent,
		},
	}
}

func (s TestNodeSpec) node() *policyhierarchy_v1.PolicyNode {
	return createTestNode(s.Name, s.Parent)
}

type TestNodeSpecs []TestNodeSpec

// Alternate implementation strategy for ancestry.
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

// Alternate implementation testing strategy fro subtree.
func (s TestNodeSpecs) Subtree(name string) []string {
	nodes := []string{name}
	for _, spec := range s {
		if spec.Parent == name {
			nodes = append(nodes, s.Subtree(spec.Name)...)
		}
	}
	return nodes
}

// Fake output type for ease of use
type TestAggregatedOutput struct {
	meta_v1.TypeMeta
	meta_v1.ObjectMeta

	Ancestry []string
}

type TestAggregatedNode struct {
	Ancestry []string
}

func (s *TestAggregatedNode) Aggregated(childNode *policyhierarchy_v1.PolicyNode) AggregatedNode {
	ancestry := make([]string, len(s.Ancestry))
	copy(ancestry, s.Ancestry)
	ancestry = append(ancestry, childNode.Name)
	return &TestAggregatedNode{Ancestry: ancestry}
}

func (s *TestAggregatedNode) Generate() Instances {
	return Instances{"test": &TestAggregatedOutput{Ancestry: s.Ancestry}}
}

func NewTestAggregatedNode() AggregatedNode {
	return &TestAggregatedNode{}
}
