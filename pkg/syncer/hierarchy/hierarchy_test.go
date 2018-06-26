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
	"testing"

	"sort"

	"github.com/google/go-cmp/cmp"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/fakeinformers"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestNode(name, parent string, nodeType policyhierarchy_v1.PolicyNodeType) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Parent: parent,
			Type:   nodeType,
		},
	}
}

func TestGetAncestry(t *testing.T) {
	validHierarchy := New(fakeinformers.NewPolicyNodeInformer(
		createTestNode("root", "", policyhierarchy_v1.Policyspace),
		createTestNode("child1", "root", policyhierarchy_v1.Policyspace),
		createTestNode("child2", "root", policyhierarchy_v1.Policyspace),
		createTestNode("child1-1", "child1", policyhierarchy_v1.Namespace),
		createTestNode("child1-2", "child1", policyhierarchy_v1.Namespace),
		createTestNode("child2-1", "child2", policyhierarchy_v1.Namespace),
	))
	invalidHierarchy := New(fakeinformers.NewPolicyNodeInformer(
		createTestNode("root", "", policyhierarchy_v1.Policyspace),
		createTestNode("child1", "root", policyhierarchy_v1.Policyspace),
		createTestNode("child2", "root", policyhierarchy_v1.Policyspace),
		createTestNode("child3-1", "child3", policyhierarchy_v1.Namespace),

		// reserved namespace in ancestry
		createTestNode("reserved1", "root", policyhierarchy_v1.ReservedNamespace),
		createTestNode("reserved-ns-1", "reserved1", policyhierarchy_v1.Policyspace),

		// Namespace in ancestry
		createTestNode("namespace-1", "root", policyhierarchy_v1.Namespace),
		createTestNode("namespace-ns-1", "namespace-1", policyhierarchy_v1.Namespace),
	))

	tests := []struct {
		name         string
		hierarchy    *Hierarchy
		nodeName     string
		wantAncestry Ancestry
		wantErr      error
	}{
		{
			name:      "leaf node",
			hierarchy: validHierarchy,
			nodeName:  "child2-1",
			wantAncestry: Ancestry{
				createTestNode("child2-1", "child2", policyhierarchy_v1.Namespace),
				createTestNode("child2", "root", policyhierarchy_v1.Policyspace),
				createTestNode("root", "", policyhierarchy_v1.Policyspace),
			},
		},
		{
			name:      "non-leaf node",
			hierarchy: validHierarchy,
			nodeName:  "child1",
			wantAncestry: Ancestry{
				createTestNode("child1", "root", policyhierarchy_v1.Policyspace),
				createTestNode("root", "", policyhierarchy_v1.Policyspace),
			},
		},
		{
			name:      "root node",
			hierarchy: validHierarchy,
			nodeName:  "root",
			wantAncestry: Ancestry{
				createTestNode("root", "", policyhierarchy_v1.Policyspace),
			},
		},
		{
			name:      "node not found",
			hierarchy: validHierarchy,
			nodeName:  "foobar",
			wantErr:   &NotFoundError{"foobar"},
		},
		{
			name:      "incomplete hierarchy",
			hierarchy: invalidHierarchy,
			nodeName:  "child3-1",
			wantErr: &ConsistencyError{
				ancestry: Ancestry{createTestNode("child3-1", "child3", policyhierarchy_v1.Namespace)},
				missing:  "child3"},
		},
		{
			name:      "reserved in ancestry",
			hierarchy: invalidHierarchy,
			nodeName:  "reserved-ns-1",
			wantErr: &ConsistencyError{
				ancestry: Ancestry{createTestNode("reserved-ns-1", "reserved1", policyhierarchy_v1.Policyspace)}},
		},
		{
			name:      "namespace in ancestry",
			hierarchy: invalidHierarchy,
			nodeName:  "namespace-ns-1",
			wantErr: &ConsistencyError{
				ancestry: Ancestry{createTestNode("namespace-ns-1", "namespace-1", policyhierarchy_v1.Namespace)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAncestry, err := tt.hierarchy.Ancestry(tt.nodeName)
			if !cmp.Equal(err, tt.wantErr, cmp.AllowUnexported(NotFoundError{}, ConsistencyError{})) {
				t.Errorf("Unexpected ancestry error, got: %v, want: %v", err, tt.wantErr)
			}
			if !cmp.Equal(gotAncestry, tt.wantAncestry) {
				t.Errorf("Ancestry generation failed got: %s, want: %s", gotAncestry, tt.wantAncestry)
			}
		})
	}
}

func TestGetSubtree(t *testing.T) {
	hierarchy := New(fakeinformers.NewPolicyNodeInformer(
		createTestNode("root", "", policyhierarchy_v1.Policyspace),
		createTestNode("child1", "root", policyhierarchy_v1.Policyspace),
		createTestNode("child2", "root", policyhierarchy_v1.Policyspace),
		createTestNode("child1-1", "child1", policyhierarchy_v1.Policyspace),
		createTestNode("child1-2", "child1", policyhierarchy_v1.Policyspace),
		createTestNode("child2-1", "child2", policyhierarchy_v1.Policyspace),
	))

	tests := []struct {
		name        string
		nodeName    string
		wantSubtree []string
		wantErr     error
	}{
		{
			name:     "leaf node",
			nodeName: "child2-1",
			wantSubtree: []string{
				"child2-1",
			},
		},
		{
			name:     "non-leaf node",
			nodeName: "child1",
			wantSubtree: []string{
				"child1",
				"child1-1",
				"child1-2",
			},
		},
		{
			name:     "root node",
			nodeName: "root",
			wantSubtree: []string{
				"root",
				"child1",
				"child1-1",
				"child1-2",
				"child2",
				"child2-1",
			},
		},
		{
			name:     "node not found",
			nodeName: "foobar",
			wantErr:  &NotFoundError{"foobar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSubtree, err := hierarchy.Subtree(tt.nodeName)
			if !cmp.Equal(err, tt.wantErr, cmp.AllowUnexported(NotFoundError{})) {
				t.Errorf("Unexpected subtree error, got: %v, want: %v", err, tt.wantErr)
			}
			sort.Strings(gotSubtree)
			sort.Strings(tt.wantSubtree)
			if !cmp.Equal(gotSubtree, tt.wantSubtree) {
				t.Errorf("Subtree generation failed got: %s, want: %s", gotSubtree, tt.wantSubtree)
			}
		})
	}
}

func TestAggregateAncestry(t *testing.T) {
	tests := []struct {
		name          string
		ancestry      Ancestry
		factory       AggregatedNodeFactory
		wantInstances Instances
		wantNode      *policyhierarchy_v1.PolicyNode
	}{
		{
			name: "empty ancestry",
			factory: func() AggregatedNode {
				return &TestAggregatedNode{
					Ancestry: []string{
						"root",
					},
				}
			},
			wantInstances: Instances{
				&TestAggregatedOutput{
					Ancestry: []string{
						"root",
					},
				},
			},
		},
		{
			name: "only root ancestry",
			ancestry: Ancestry{
				createTestNode("child", "root", policyhierarchy_v1.Policyspace),
			},
			factory: func() AggregatedNode {
				return &TestAggregatedNode{
					Ancestry: []string{
						"root",
					},
				}
			},
			wantInstances: Instances{
				&TestAggregatedOutput{
					Ancestry: []string{
						"root",
						"child",
					},
				},
			},
			wantNode: createTestNode("child", "root", policyhierarchy_v1.Policyspace),
		},
		{
			name: "extended ancestry",
			ancestry: Ancestry{
				createTestNode("child5", "child4", policyhierarchy_v1.Namespace),
				createTestNode("child4", "child3", policyhierarchy_v1.Policyspace),
				createTestNode("child3", "child2", policyhierarchy_v1.Policyspace),
				createTestNode("child2", "child1", policyhierarchy_v1.Policyspace),
				createTestNode("child1", "root", policyhierarchy_v1.Policyspace),
			},
			factory: func() AggregatedNode {
				return &TestAggregatedNode{
					Ancestry: []string{
						"root",
					},
				}
			},
			wantNode: createTestNode("child5", "child4", policyhierarchy_v1.Namespace),
			wantInstances: Instances{
				&TestAggregatedOutput{
					Ancestry: []string{
						"root",
						"child1",
						"child2",
						"child3",
						"child4",
						"child5",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInstances := tt.ancestry.Aggregate(tt.factory)
			if !cmp.Equal(gotInstances, tt.wantInstances) {
				t.Errorf("Ancestry aggregation failed got: %s, want: %s", gotInstances, tt.wantInstances)
			}
			if tt.wantNode != nil {
				gotNode := tt.ancestry.Node()
				if !cmp.Equal(gotNode, tt.wantNode) {
					t.Errorf("Ancestry requested node is incorrect got: %s, want: %s", gotInstances, tt.wantInstances)
				}
			}
		})
	}
}
