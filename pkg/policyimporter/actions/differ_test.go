// Reviewed by sunilarora
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

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"k8s.io/api/extensions/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testCase struct {
	testName                           string
	oldNodes, newNodes                 []v1.PolicyNode
	oldClusterPolicy, newClusterPolicy *v1.ClusterPolicy
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
			testName: "One node Create",
			oldNodes: []v1.PolicyNode{},
			newNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			expected: []string{"nomos.dev/v1/PolicyNodes/r/upsert"},
		},
		{
			testName: "One node delete",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			newNodes: []v1.PolicyNode{},
			expected: []string{"nomos.dev/v1/PolicyNodes/r/delete"},
		},
		{
			testName: "Rename root node",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r2", ""),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/r2/upsert",
				"nomos.dev/v1/PolicyNodes/r/delete",
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
				"nomos.dev/v1/PolicyNodes/c1/upsert",
				"nomos.dev/v1/PolicyNodes/c2/upsert",
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
				"nomos.dev/v1/PolicyNodes/c1/upsert",
				"nomos.dev/v1/PolicyNodes/c2/upsert",
				"nomos.dev/v1/PolicyNodes/co2/delete",
				"nomos.dev/v1/PolicyNodes/co1/delete",
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
				"nomos.dev/v1/PolicyNodes/c3/upsert",
				"nomos.dev/v1/PolicyNodes/co2/upsert",
				"nomos.dev/v1/PolicyNodes/co1/delete",
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
				"nomos.dev/v1/PolicyNodes/c3/upsert",
				"nomos.dev/v1/PolicyNodes/c3.1/upsert",
				"nomos.dev/v1/PolicyNodes/c2/upsert",
				"nomos.dev/v1/PolicyNodes/c1/delete",
			},
		},
		{
			testName:         "ClusterPolicy create",
			newClusterPolicy: clusterPolicy("foo", true),
			expected: []string{
				"nomos.dev/v1/ClusterPolicies/foo/upsert",
			},
		},
		{
			testName:         "ClusterPolicy update",
			oldClusterPolicy: clusterPolicy("foo", true),
			newClusterPolicy: clusterPolicy("foo", false),
			expected: []string{
				"nomos.dev/v1/ClusterPolicies/foo/upsert",
			},
		},
		{
			testName:         "ClusterPolicy update no change",
			oldClusterPolicy: clusterPolicy("foo", true),
			newClusterPolicy: clusterPolicy("foo", true),
			expected:         []string{},
		},
		{
			testName:         "ClusterPolicy delete",
			oldClusterPolicy: clusterPolicy("foo", true),
			expected: []string{
				"nomos.dev/v1/ClusterPolicies/foo/delete",
			},
		},
		{
			testName: "Create 2 nodes and a ClusterPolicy",
			oldNodes: []v1.PolicyNode{
				policyNode("r", ""),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", ""),
				policyNode("c2", "c1"),
				policyNode("c1", "r"),
			},
			newClusterPolicy: clusterPolicy("foo", true),
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c1/upsert",
				"nomos.dev/v1/PolicyNodes/c2/upsert",
				"nomos.dev/v1/ClusterPolicies/foo/upsert",
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			g := NewDiffer(NewPolicyNodeActionSpec(nil, nil), NewClusterPolicyActionSpec(nil, nil))

			actual := g.Diff(allPolicies(test.oldNodes, test.oldClusterPolicy), allPolicies(test.newNodes, test.newClusterPolicy))

			if len(actual) != len(test.expected) {
				t.Fatalf("Actual number of actions was %d but expected %d",
					len(actual), len(test.expected))
			}

			for aIdx, action := range actual {
				if action.String() != test.expected[aIdx] {
					t.Fatalf("Actual action at index %d was %q but expected %q",
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

func clusterPolicy(name string, priviledged bool) *v1.ClusterPolicy {
	return &v1.ClusterPolicy{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.ClusterPolicySpec{
			Policies: v1.ClusterPolicies{
				PodSecurityPoliciesV1Beta1: []v1beta1.PodSecurityPolicy{
					{Spec: v1beta1.PodSecurityPolicySpec{Privileged: priviledged}}},
			},
		},
	}
}

func allPolicies(nodes []v1.PolicyNode, clusterPolicy *v1.ClusterPolicy) v1.AllPolicies {
	p := v1.AllPolicies{
		PolicyNodes:   make(map[string]v1.PolicyNode),
		ClusterPolicy: clusterPolicy,
	}
	for _, n := range nodes {
		p.PolicyNodes[n.Name] = n
	}
	return p
}
