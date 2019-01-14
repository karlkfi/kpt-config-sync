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
	"github.com/google/nomos/pkg/util/policynode/validator"
	"k8s.io/api/extensions/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				policyNode("r", "", v1.Policyspace),
			},
			expected: []string{"nomos.dev/v1/PolicyNodes/r/create"},
		},
		{
			testName: "One node delete",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{},
			expected: []string{"nomos.dev/v1/PolicyNodes/r/delete"},
		},
		{
			testName: "Rename root node",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r2", "", v1.Policyspace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/r2/create",
				"nomos.dev/v1/PolicyNodes/r/delete",
			},
		},
		{
			testName: "Rename policyspace with children",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "c1", v1.Namespace),
				policyNode("c3", "c1", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c4", "r", v1.Policyspace),
				policyNode("c2", "c4", v1.Namespace),
				policyNode("c3", "c4", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c4/create",
				"nomos.dev/v1/PolicyNodes/c2/update",
				"nomos.dev/v1/PolicyNodes/c3/update",
				"nomos.dev/v1/PolicyNodes/c1/delete",
			},
		},
		{
			testName: "Create 2 nodes",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "c1", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c1/create",
				"nomos.dev/v1/PolicyNodes/c2/create",
			},
		},
		{
			testName: "Create 2 nodes and delete 2",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("co1", "r", v1.Policyspace),
				policyNode("co2", "co1", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c2", "c1", v1.Namespace),
				policyNode("c1", "r", v1.Policyspace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c1/create",
				"nomos.dev/v1/PolicyNodes/c2/create",
				"nomos.dev/v1/PolicyNodes/co2/delete",
				"nomos.dev/v1/PolicyNodes/co1/delete",
			},
		},
		{
			testName: "Move a grandchild under root and create a new child under it",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("co1", "r", v1.Policyspace),
				policyNode("co2", "co1", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("co2", "r", v1.Policyspace),
				policyNode("c3", "co2", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c3/create",
				"nomos.dev/v1/PolicyNodes/co2/update",
				"nomos.dev/v1/PolicyNodes/co1/delete",
			},
		},
		{
			testName: "Re-parent namespace node",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c3", "c1", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c3", "c2", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c3/update",
			},
		},
		{
			testName: "Re-parent policyspace node",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "c1", v1.Policyspace),
				policyNode("c3", "c2", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c3", "c2", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c2/update",
			},
		},
		{
			testName: "Re-parent and delete",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "c1", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c3", "r", v1.Policyspace),
				policyNode("c3.1", "c3", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c3/create",
				"nomos.dev/v1/PolicyNodes/c3.1/create",
				"nomos.dev/v1/PolicyNodes/c2/update",
				"nomos.dev/v1/PolicyNodes/c1/delete",
			},
		},
		{
			testName: "Swap nodes with parent-child relationship",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "c1", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c1", "c2", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c2/update",
				"nomos.dev/v1/PolicyNodes/c1/update",
			},
		},
		{
			testName: "Swap nodes with parent-child relationship and create grandchild",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "c1", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c1", "c2", v1.Policyspace),
				policyNode("c3", "c1", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c3/create",
				"nomos.dev/v1/PolicyNodes/c2/update",
				"nomos.dev/v1/PolicyNodes/c1/update",
			},
		},
		{
			testName: "Swap namespace child with aunt policyspace",
			oldNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c1", "r", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c3", "c1", v1.Namespace),
				policyNode("c4", "c2", v1.Namespace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c4", "r", v1.Policyspace),
				policyNode("c2", "r", v1.Policyspace),
				policyNode("c3", "c4", v1.Namespace),
				policyNode("c1", "c2", v1.Namespace),
			},
			expected: []string{
				"nomos.dev/v1/PolicyNodes/c4/update",
				"nomos.dev/v1/PolicyNodes/c1/update",
				"nomos.dev/v1/PolicyNodes/c3/update",
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
				policyNode("r", "", v1.Policyspace),
			},
			newNodes: []v1.PolicyNode{
				policyNode("r", "", v1.Policyspace),
				policyNode("c2", "c1", v1.Namespace),
				policyNode("c1", "r", v1.Policyspace),
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
				makeSync("", "ResourceQuota", "v1"),
			},
			expected: []string{
				"nomos.dev/v1alpha1/Syncs/ResourceQuota/create",
			},
		},
		{
			testName: "Sync update",
			oldSyncs: []v1alpha1.Sync{
				makeSync("", "ResourceQuota", "v1"),
			},
			newSyncs: []v1alpha1.Sync{
				makeSync("", "ResourceQuota", "v2"),
			},
			expected: []string{
				"nomos.dev/v1alpha1/Syncs/ResourceQuota/update",
			},
		},
		{
			testName: "Sync update no change",
			oldSyncs: []v1alpha1.Sync{
				makeSync("", "ResourceQuota", "v1"),
			},
			newSyncs: []v1alpha1.Sync{
				makeSync("", "ResourceQuota", "v1"),
			},
			expected: []string{},
		},
		{
			testName: "Sync delete",
			oldSyncs: []v1alpha1.Sync{
				makeSync("", "ResourceQuota", "v1"),
			},
			newSyncs: []v1alpha1.Sync{},
			expected: []string{}, // deletes are exposed through SyncDeletes method, not Diff.
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
			if err := validate(policyNodes); err != nil {
				t.Errorf("Policy hierarchy state became invalid after executing actions")
			}
		})
	}
}

func TestSyncDeletes(t *testing.T) {
	g := NewDiffer(NewFactories(nil, nil, nil, nil, nil))
	g.SortDiff = true

	actual := g.SyncDeletes(
		map[string]v1alpha1.Sync{
			"ResourceQuota": makeSync("", "ResourceQuota", "v1"),
		},
		map[string]v1alpha1.Sync{})
	expected := []string{"ResourceQuota"}
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Actual and expected deletes didn't match: %#v", diff)
	}
}

type reducerTestCase struct {
	testName         string
	current, desired map[string]v1alpha1.Sync
	expected         []v1alpha1.Sync
}

func TestSyncReductions(t *testing.T) {
	g := NewDiffer(NewFactories(nil, nil, nil, nil, nil))
	g.SortDiff = true

	for _, test := range []reducerTestCase{
		{
			testName: "Empty set",
			current:  map[string]v1alpha1.Sync{},
			desired:  map[string]v1alpha1.Sync{},
			expected: nil,
		},
		{
			testName: "One version, no change",
			current: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1"),
			},
			desired: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1"),
			},
			expected: nil,
		},
		{
			testName: "Delete one version",
			current: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1", "v2"),
			},
			desired: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1"),
			},
			expected: []v1alpha1.Sync{
				makeSync("", "ResourceQuota", "v1"),
			},
		},
		{
			testName: "Delete one version, add one version",
			current: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1", "v2"),
			},
			desired: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1", "v3"),
			},
			expected: []v1alpha1.Sync{
				makeSync("", "ResourceQuota", "v1"),
			},
		},
		{
			testName: "Add one version",
			current: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1"),
			},
			desired: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1", "v3"),
			},
			expected: nil,
		},
		{
			testName: "Only ordering changes",
			current: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v1", "v2"),
			},
			desired: map[string]v1alpha1.Sync{
				"ResourceQuota": makeSync("", "ResourceQuota", "v2", "v1"),
			},
			expected: nil,
		},
		{
			testName: "Delete one kind",
			current: map[string]v1alpha1.Sync{
				"Role": {
					TypeMeta: meta.TypeMeta{
						APIVersion: "nomos.dev/v1alpha1",
						Kind:       "Sync",
					},
					ObjectMeta: meta.ObjectMeta{
						Name: "Role",
					},
					Spec: v1alpha1.SyncSpec{
						Groups: []v1alpha1.SyncGroup{
							{
								Group: "rbac.authorization.k8s.io",
								Kinds: []v1alpha1.SyncKind{
									{
										Kind: "Role",
										Versions: []v1alpha1.SyncVersion{
											{
												Version: "v1",
											},
										},
									},
									{
										Kind: "RoleBinding",
										Versions: []v1alpha1.SyncVersion{
											{
												Version: "v2",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			desired: map[string]v1alpha1.Sync{
				"Role": makeSync("rbac.authorization.k8s.io", "Role", "v1"),
			},
			expected: []v1alpha1.Sync{
				makeSync("rbac.authorization.k8s.io", "Role", "v1"),
			},
		},
		{
			testName: "Delete one group",
			current: map[string]v1alpha1.Sync{
				"Role": {
					TypeMeta: meta.TypeMeta{
						APIVersion: "nomos.dev/v1alpha1",
						Kind:       "Sync",
					},
					ObjectMeta: meta.ObjectMeta{
						Name: "Role",
					},
					Spec: v1alpha1.SyncSpec{
						Groups: []v1alpha1.SyncGroup{
							{
								Group: "rbac.authorization.k8s.io",
								Kinds: []v1alpha1.SyncKind{
									{
										Kind: "Role",
										Versions: []v1alpha1.SyncVersion{
											{
												Version: "v1",
											},
										},
									},
								},
							},
							{
								Kinds: []v1alpha1.SyncKind{
									{
										Kind: "ResourceQuota",
										Versions: []v1alpha1.SyncVersion{
											{
												Version: "v2",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			desired: map[string]v1alpha1.Sync{
				"Role": makeSync("rbac.authorization.k8s.io", "Role", "v1"),
			},
			expected: []v1alpha1.Sync{
				makeSync("rbac.authorization.k8s.io", "Role", "v1"),
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			actual := g.SyncReductions(test.current, test.desired)
			if diff := cmp.Diff(actual, test.expected); diff != "" {
				t.Errorf("Actual and expected reductions didn't match: %#v", diff)
			}
		})
	}
}

func validate(nodes map[string]v1.PolicyNode) error {
	v := validator.New()
	v.AllowMultipleRoots = true
	for _, n := range nodes {
		if err := v.Add(&n); err != nil {
			return err
		}
	}
	return v.Validate()
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
		nodes[a.Name()] = policyNode(a.Name(), pn.Spec.Parent, pn.Spec.Type)
	case action.UpdateOperation:
		upAct := a.(*action.ReflectiveUpdateAction)
		old := nodes[upAct.Resource()]
		r, err := upAct.UpdatedResource(&old)
		if err != nil {
			t.Fatalf("Failed to update resource: %v", err)
		}
		pn := r.(*v1.PolicyNode)
		nodes[a.Name()] = policyNode(a.Name(), pn.Spec.Parent, pn.Spec.Type)
	case action.UpsertOperation:
		r := a.(*action.ReflectiveUpsertAction).UpsertedResouce()
		pn := r.(*v1.PolicyNode)
		nodes[a.Name()] = policyNode(a.Name(), pn.Spec.Parent, pn.Spec.Type)
	case action.DeleteOperation:
		delete(nodes, a.Name())
	default:
		t.Fatalf("Unexpected operation: %v", op)
	}
}

func policyNode(name, parent string, t v1.PolicyNodeType) v1.PolicyNode {
	return v1.PolicyNode{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.PolicyNodeSpec{
			Parent: parent,
			Type:   t,
		},
	}
}

func clusterPolicy(name string, priviledged bool) *v1.ClusterPolicy {
	return &v1.ClusterPolicy{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.ClusterPolicySpec{
			PodSecurityPoliciesV1Beta1: []v1beta1.PodSecurityPolicy{
				{Spec: v1beta1.PodSecurityPolicySpec{Privileged: priviledged}}},
		},
	}
}

func makeSync(group, kind string, versions ...string) v1alpha1.Sync {
	var svs []v1alpha1.SyncVersion
	for _, version := range versions {
		sv := v1alpha1.SyncVersion{Version: version}
		svs = append(svs, sv)
	}
	return v1alpha1.Sync{
		TypeMeta: meta.TypeMeta{
			APIVersion: "nomos.dev/v1alpha1",
			Kind:       "Sync",
		},
		ObjectMeta: meta.ObjectMeta{
			Name: kind,
		},
		Spec: v1alpha1.SyncSpec{
			Groups: []v1alpha1.SyncGroup{
				{
					Group: group,
					Kinds: []v1alpha1.SyncKind{
						{
							Kind:     kind,
							Versions: svs,
						},
					},
				},
			},
		},
	}
}

func allPolicies(nodes []v1.PolicyNode, clusterPolicy *v1.ClusterPolicy, syncs []v1alpha1.Sync) v1.AllPolicies {
	policies := v1.AllPolicies{
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
