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
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type generatorTestCase struct {
	testName string
	oldNodes []v1.PolicyNode
	newNodes []v1.PolicyNode
	// String representation of expected actions
	expected []string
}

func TestGenerator(t *testing.T) {
	for _, test := range []generatorTestCase{
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
			expected: []string{"policynode.r.upsert"},
		},
		{
			testName: "One delete",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			newNodes: []v1.PolicyNode{},
			expected: []string{"policynode.r.delete"},
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
				"policynode.r2.upsert",
				"policynode.r.delete",
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
				"policynode.c1.upsert",
				"policynode.c2.upsert",
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
				"policynode.c1.upsert",
				"policynode.c2.upsert",
				"policynode.co2.delete",
				"policynode.co1.delete",
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
				"policynode.c3.upsert",
				"policynode.co2.upsert",
				"policynode.co1.delete",
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
				"policynode.c3.upsert",
				"policynode.c3.1.upsert",
				"policynode.c2.upsert",
				"policynode.c1.delete",
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			g := NewGenerator(test.oldNodes, test.newNodes, nil, nil)

			actual := g.GenerateActions()

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
