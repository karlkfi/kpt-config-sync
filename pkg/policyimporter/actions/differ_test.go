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
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/util/policynode"
	"k8s.io/api/policy/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type testCase struct {
	testName                           string
	oldNodes, newNodes                 []v1.PolicyNode
	oldClusterPolicy, newClusterPolicy *v1.ClusterPolicy
	oldSyncs, newSyncs                 []v1alpha1.Sync
	// String representation of expected actions
	expected []string
}

func TestDiffer(t *testing.T) {
	for _, test := range []testCase{
		{
			testName: "Nil",
			expected: []string{},
		},
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
				policyNode("r", v1.Policyspace),
			},
			expected: []string{"nomos.dev/v1/PolicyNodes/r/create"},
		},
		{
			testName: "One node delete",
			oldNodes: []v1.PolicyNode{
				policyNode("r", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{},
			expected: []string{"nomos.dev/v1/PolicyNodes/r/delete"},
		},
		{
			testName: "Rename root node",
			oldNodes: []v1.PolicyNode{
				policyNode("r", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r2", v1.Policyspace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/r2/create",
				"nomos.dev/v1/PolicyNodes/r/delete",
			},
		},
		{
			testName: "Create 2 nodes",
			oldNodes: []v1.PolicyNode{
				policyNode("r", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", v1.Policyspace),
				policyNode("c1", v1.Policyspace),
				policyNode("c2", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c1/create",
				"nomos.dev/v1/PolicyNodes/c2/create",
			},
		},
		{
			testName: "Create 2 nodes and delete 2",
			oldNodes: []v1.PolicyNode{
				policyNode("r", v1.Policyspace),
				policyNode("co1", v1.Policyspace),
				policyNode("co2", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", v1.Policyspace),
				policyNode("c2", v1.Namespace),
				policyNode("c1", v1.Policyspace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c1/create",
				"nomos.dev/v1/PolicyNodes/c2/create",
				"nomos.dev/v1/PolicyNodes/co2/delete",
				"nomos.dev/v1/PolicyNodes/co1/delete",
			},
		},
		{
			testName:         "ClusterPolicy create",
			newClusterPolicy: clusterPolicy("foo", true),
			expected: []string{
				"nomos.dev/v1/ClusterPolicies/foo/create",
			},
		},
		{
			testName:         "ClusterPolicy update",
			oldClusterPolicy: clusterPolicy("foo", true),
			newClusterPolicy: clusterPolicy("foo", false),
			expected: []string{
				"nomos.dev/v1/ClusterPolicies/foo/update",
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
				policyNode("r", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", v1.Policyspace),
				policyNode("c2", v1.Namespace),
				policyNode("c1", v1.Policyspace),
			},
			newClusterPolicy: clusterPolicy("foo", true),
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c1/create",
				"nomos.dev/v1/PolicyNodes/c2/create",
				"nomos.dev/v1/ClusterPolicies/foo/create",
			},
		},
		{
			testName: "Empty Syncs",
			oldSyncs: []v1alpha1.Sync{},
			newSyncs: []v1alpha1.Sync{},
			expected: []string{},
		},
		{
			testName: "Sync create",
			oldSyncs: []v1alpha1.Sync{},
			newSyncs: []v1alpha1.Sync{
				*v1alpha1.NewSync("", "ResourceQuota"),
			},
			expected: []string{
				"nomos.dev/v1alpha1/Syncs/resourcequota/create",
			},
		},
		{
			testName: "Sync update no change",
			oldSyncs: []v1alpha1.Sync{
				*v1alpha1.NewSync("", "ResourceQuota"),
			},
			newSyncs: []v1alpha1.Sync{
				*v1alpha1.NewSync("", "ResourceQuota"),
			},
			expected: []string{},
		},
		{
			testName: "Sync delete",
			oldSyncs: []v1alpha1.Sync{
				*v1alpha1.NewSync("", "ResourceQuota"),
			},
			newSyncs: []v1alpha1.Sync{},
			expected: []string{
				"nomos.dev/v1alpha1/Syncs/resourcequota/delete",
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			g := NewDiffer(NewFactories(nil, nil, nil, nil, nil))
			g.SortDiff = true

			gotActions := g.Diff(
				allPolicies(test.oldNodes, test.oldClusterPolicy, test.oldSyncs),
				allPolicies(test.newNodes, test.newClusterPolicy, test.newSyncs))

			if len(gotActions) != len(test.expected) {
				t.Fatalf("Actual number of actions was %d but expected %d",
					len(gotActions), len(test.expected))
			}

			actual := []string{}
			for _, a := range gotActions {
				actual = append(actual, a.String())
			}
			sort.Strings(test.expected)
			sort.Strings(actual)
			if !cmp.Equal(test.expected, actual) {
				t.Fatalf("Exepcted and actual actions differ: %s", cmp.Diff(test.expected, actual))
			}

			policyNodes := make(map[string]v1.PolicyNode)
			for _, pn := range test.oldNodes {
				policyNodes[pn.Name] = pn
			}
			for _, action := range gotActions {
				executeAction(t, action, policyNodes)
			}
		})
	}
}

func executeAction(t *testing.T, a action.Interface, nodes map[string]v1.PolicyNode) {
	if a.Kind() != action.Plural(v1.PolicyNode{}) {
		// We only test transient states for PolicyNodes
		return
	}
	op := a.Operation()
	switch op {
	case action.CreateOperation:
		r := a.(*action.ReflectiveCreateAction).Item()
		pn := r.(*v1.PolicyNode)
		nodes[a.Name()] = policyNode(a.Name(), pn.Spec.Type)
	case action.UpdateOperation:
		upAct := a.(*action.ReflectiveUpdateAction)
		old := nodes[upAct.Resource()]
		r, err := upAct.UpdatedResource(&old)
		if err != nil {
			t.Fatalf("Failed to update resource: %v", err)
		}
		pn := r.(*v1.PolicyNode)
		nodes[a.Name()] = policyNode(a.Name(), pn.Spec.Type)
	case action.UpsertOperation:
		r := a.(*action.ReflectiveUpsertAction).UpsertedResouce()
		pn := r.(*v1.PolicyNode)
		nodes[a.Name()] = policyNode(a.Name(), pn.Spec.Type)
	case action.DeleteOperation:
		delete(nodes, a.Name())
	default:
		t.Fatalf("Unexpected operation: %v", op)
	}
}

func policyNode(name string, t v1.PolicyNodeType) v1.PolicyNode {
	return v1.PolicyNode{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.PolicyNodeSpec{
			Type: t,
		},
	}
}

func clusterPolicy(name string, priviledged bool) *v1.ClusterPolicy {
	return &v1.ClusterPolicy{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.ClusterPolicySpec{
			Resources: []v1.GenericResources{
				{
					Group: v1beta1.GroupName,
					Kind:  "PodSecurityPolicy",
					Versions: []v1.GenericVersionResources{
						{
							Version: v1beta1.SchemeGroupVersion.Version,
							Objects: []runtime.RawExtension{
								{
									Object: &v1beta1.PodSecurityPolicy{
										Spec: v1beta1.PodSecurityPolicySpec{Privileged: priviledged},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func allPolicies(nodes []v1.PolicyNode, clusterPolicy *v1.ClusterPolicy, syncs []v1alpha1.Sync) policynode.AllPolicies {
	policies := policynode.AllPolicies{
		ClusterPolicy: clusterPolicy,
	}

	for i, n := range nodes {
		if i == 0 {
			policies.PolicyNodes = make(map[string]v1.PolicyNode)
		}
		policies.PolicyNodes[n.Name] = n
	}

	if len(syncs) > 0 {
		policies.Syncs = make(map[string]v1alpha1.Sync)
	}
	for _, s := range syncs {
		policies.Syncs[s.Name] = s
	}

	return policies
}
