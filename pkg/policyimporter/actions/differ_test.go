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

package actions

import (
	"testing"

	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testCase struct {
	testName string
	oldNodes []v1.PolicyNode
	newNodes []v1.PolicyNode
	// String representation of expected actions
	expected []string
}

func TestDiffer(t *testing.T) {
	for _, test := range []testCase{
		{
			testName: "All Empty",
			oldNodes: []v1.PolicyNode{},
			newNodes: []v1.PolicyNode{},
			expected: []string{},
		},
		{
			testName: "One Create",
			oldNodes: []v1.PolicyNode{},
			newNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			expected: []string{"PolicyNodes.r.upsert"},
		},
		{
			testName: "One delete",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			newNodes: []v1.PolicyNode{},
			expected: []string{"PolicyNodes.r.delete"},
		},
		{
			testName: "Rename root",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r2", ""),
			},
			expected: []string{
				"PolicyNodes.r2.upsert",
				"PolicyNodes.r.delete",
			},
		},
		{
			testName: "Create 2 nodes",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("c2", "c1"),
				policyNode("c1", "r"),
			},
			expected: []string{
				"PolicyNodes.c1.upsert",
				"PolicyNodes.c2.upsert",
			},
		},
		{
			testName: "Create 2 nodes and delete 2",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("co1", "r"),
				policyNode("co2", "co1"),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("c2", "c1"),
				policyNode("c1", "r"),
			},
			expected: []string{
				"PolicyNodes.c1.upsert",
				"PolicyNodes.c2.upsert",
				"PolicyNodes.co2.delete",
				"PolicyNodes.co1.delete",
			},
		},
		{
			testName: "Move a grandchild under root and create a new child under it",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("co1", "r"),
				policyNode("co2", "co1"),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("co2", "r"),
				policyNode("c3", "co2"),
			},
			expected: []string{
				"PolicyNodes.c3.upsert",
				"PolicyNodes.co2.upsert",
				"PolicyNodes.co1.delete",
			},
		},
		{
			testName: "Re-parent and delete",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("c1", "r"),
				policyNode("c2", "c1"),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("c2", "r"),
				policyNode("c3", "r"),
				policyNode("c3.1", "c3"),
			},
			expected: []string{
				"PolicyNodes.c3.upsert",
				"PolicyNodes.c3.1.upsert",
				"PolicyNodes.c2.upsert",
				"PolicyNodes.c1.delete",
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			g := NewDiffer(NewPolicyNodeActionSpec(nil, nil), nil)

			actual := g.Diff(allPolicies(test.oldNodes), allPolicies(test.newNodes))

			if len(actual) != len(test.expected) {
				t.Fatalf("Unexpected number of actions was %d but expected %d",
					len(actual), len(test.expected))
			}

			for aIdx, action := range actual {
				if action.String() != test.expected[aIdx] {
					t.Fatalf("Unexpected action at index %d was %q but expected %q",
						aIdx, action.String(), test.expected[aIdx])
				}
			}
			// TODO: Consider adding tests that ensure invariants after each action. The tests
			// currently only enforce it by the careful checking of ordering by the test writer.
		})
	}
}

func policyNode(name, parent string) v1.PolicyNode {
	return v1.PolicyNode{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.PolicyNodeSpec{
			Parent: parent,
		},
	}
}

func allPolicies(nodes []v1.PolicyNode) v1.AllPolicies {
	p := v1.AllPolicies{
		PolicyNodes: make(map[string]v1.PolicyNode),
	}
	for _, n := range nodes {
		p.PolicyNodes[n.Name] = n
	}
	return p
}
