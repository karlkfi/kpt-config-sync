/*
Copyright 2018 The Stolos Authors.
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

package flattening

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/testing/rbactesting"
	"github.com/google/nomos/pkg/util/set/stringset"
	rbac "k8s.io/api/rbac/v1"
)

func TestPolicyNode(t *testing.T) {
	tree := NewPolicyTree()
	tree.Upsert("Node", "", false, Policy{})
	if _, err := tree.Lookup("Node"); err != nil {
		t.Errorf("Node not found, but should be: %v", err)
	}
}

func TestRoots(t *testing.T) {
	type nodeParent struct {
		node, parent string
	}
	tests := []struct {
		upserts       []nodeParent
		expectedRoots []string
	}{
		{
			upserts:       []nodeParent{},
			expectedRoots: []string{},
		},
		{
			upserts: []nodeParent{
				nodeParent{"one", ""},
			},
			expectedRoots: []string{"one"},
		},
		{
			upserts: []nodeParent{
				nodeParent{"one", "two"},
			},
			expectedRoots: []string{"one"},
		},
		{
			upserts: []nodeParent{
				nodeParent{"one", "two"},
				nodeParent{"three", "four"},
			},
			expectedRoots: []string{"one", "three"},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			tree := NewPolicyTree()
			for _, upsert := range tt.upserts {
				tree.Upsert(upsert.node, upsert.parent, false, *NewPolicy())
			}
			expectedRoots := stringset.NewFromSlice(tt.expectedRoots)
			actualRoots := stringset.NewFromSlice(tree.Roots())
			if !expectedRoots.Equals(actualRoots) {
				t.Errorf("Expected: %v, actual: %v", expectedRoots, actualRoots)
			}

		})
	}
}

func TestParentOf(t *testing.T) {
	type parentChild struct {
		parent, child string
	}
	tests := []struct {
		upserts []parentChild
	}{
		{
			[]parentChild{
				{child: "1", parent: "100"},
			},
		},
		{
			[]parentChild{
				{child: "1", parent: "100"},
				{child: "2", parent: "100"},
				{child: "3", parent: "100"},
				{child: "4", parent: "100"},
				{child: "5", parent: "100"},
				{child: "6", parent: "100"},
				{child: "7", parent: "100"},
				{child: "8", parent: "100"},
				{child: "9", parent: "100"},
			},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case-%v", i), func(t *testing.T) {
			tree := NewPolicyTree()
			for _, upsert := range tt.upserts {
				tree.Upsert(upsert.child, upsert.parent, false, Policy{})
			}
			for _, upsert := range tt.upserts {
				parent, err := tree.ParentOf(upsert.child)
				if err != nil {
					t.Errorf("for upsert: %v: %v", upsert, err)
				}
				if upsert.parent != parent {
					t.Errorf("Parent mismatch: child=%v, expected=%v, actual=%v",
						upsert.child, upsert.parent, parent)
				}
			}
		})
	}
}

// role is a quick role definition for tests where most of the role content
// does not mattter.
func role(user string, verb string, namespace string) rbac.Role {
	return rbactesting.NamespaceRole(
		user,
		namespace,
		[]rbac.PolicyRule{
			rbactesting.PolicyRule(
				[]string{"*"}, []string{verb}, []string{}),
		})
}

func roleBinding(roleName, namespace string, subjects ...string) rbac.RoleBinding {
	return rbactesting.NamespaceRoleBinding("", namespace, roleName, subjects...)
}

type upsertParams struct {
	name   string
	parent string
	policy Policy
}

func TestDelete(t *testing.T) {
	tests := []struct {
		testName string
		// These are the upsert commands that will be executed.
		upserts []upsertParams
		// The name of the node to delete.
		node string
		// The nodes that remain.
		expectedRemaining *stringset.StringSet
	}{
		{
			testName: "Delete a leaf",
			upserts: []upsertParams{
				upsertParams{"One", "",
					*NewPolicy().AddRoleBinding(
						rbactesting.RoleBinding("reader", "User:jane"),
					),
				},
				upsertParams{"Two", "One",
					*NewPolicy().AddRoleBinding(
						rbactesting.RoleBinding("reader", "User:joe"),
					),
				},
				upsertParams{"Three", "Two",
					*NewPolicy().AddRoleBinding(
						rbactesting.RoleBinding("reader", "User:jessie"),
					),
				},
			},
			node:              "Three",
			expectedRemaining: stringset.NewFromSlice([]string{"One", "Two"}),
		},
		{
			testName: "Delete a middle node",
			upserts: []upsertParams{
				upsertParams{"One", "",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jane"),
					),
				},
				upsertParams{"Two", "One",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:joe"),
					),
				},
				upsertParams{"Three", "Two",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jessie"),
					),
				},
			},
			node:              "Two",
			expectedRemaining: stringset.NewFromSlice([]string{"One"}),
		},
		{
			testName: "Delete a root",
			upserts: []upsertParams{
				upsertParams{"One", "",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jane"),
					),
				},
				upsertParams{"Two", "One",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:joe"),
					),
				},
				upsertParams{"Three", "Two",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jessie"),
					),
				},
			},
			node:              "One",
			expectedRemaining: stringset.New(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			tree := NewPolicyTree()
			for _, upsert := range tt.upserts {
				tree.Upsert(upsert.name, upsert.parent, false, upsert.policy)
			}
			tree.Delete(tt.node)

			names := stringset.New()
			for node, _ := range tree.nodesByName {
				names.Add(string(node))
			}
			if !names.Equals(tt.expectedRemaining) {
				t.Errorf("Mismatch: expected: %v, actual: %v",
					tt.expectedRemaining, names)
			}
		})
	}
}

func TestFlattenRoleBindings(t *testing.T) {
	tests := []struct {
		testName string
		// These are the upsert commands that will be executed.
		upserts []upsertParams
		// The name of the node to look up at the end.
		node string
		// The expected resulting policy for the node named above.
		expected Policy
	}{
		{
			testName: "nil values",
			upserts: []upsertParams{
				upsertParams{"One", "", *NewPolicy()},
				upsertParams{"Two", "One", *NewPolicy()},
				upsertParams{"Three", "Two", *NewPolicy()},
			},
			node:     "Three",
			expected: *NewPolicy(),
		},
		{
			testName: "Upsert RoleBindings in order.",
			upserts: []upsertParams{
				upsertParams{
					"One", "",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jane"),
					),
				},
				upsertParams{"Two", "One",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:joe"),
					),
				},
				upsertParams{"Three", "Two",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jessie"),
					),
				},
			},
			node: "Three",
			expected: *NewPolicy().SetRoleBindings(
				roleBinding("reader", "Three", "User:jane"),
				roleBinding("reader", "Three", "User:joe"),
				roleBinding("reader", "Three", "User:jessie"),
			),
		},
		{
			testName: "Basic Upsert RoleBindings out of order",
			upserts: []upsertParams{
				upsertParams{"Two", "One",
					*NewPolicy().SetRoleBindings(roleBinding("", "Two")),
				},
				upsertParams{"One", "",
					*NewPolicy().SetRoleBindings(roleBinding("", "One")),
				},
			},
			node: "Two",
			expected: *NewPolicy().SetRoleBindings(
				roleBinding("", "Two"),
				roleBinding("", "Two"),
			),
		},
		{
			testName: "Upsert out of order",
			upserts: []upsertParams{
				upsertParams{"Three", "Two",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jessie"),
					),
				},
				upsertParams{"Two", "One",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:joe"),
					),
				},
				upsertParams{"One", "",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jane"),
					),
				},
			},
			node: "Three",
			expected: *NewPolicy().SetRoleBindings(
				roleBinding("reader", "Three", "User:jane"),
				roleBinding("reader", "Three", "User:joe"),
				roleBinding("reader", "Three", "User:jessie"),
			),
		},
		{
			testName: "Reparenting",
			upserts: []upsertParams{
				upsertParams{"Three", "Two", Policy{}},
				upsertParams{"Two", "One",
					*NewPolicy().SetRoleBindings(
						roleBinding("reader", "Two", "User:joe"),
					),
				},
				// "Three" is parented away from "Two" and onto "One". Also,
				// it got one policy added.
				upsertParams{"Three", "One",
					*NewPolicy().SetRoleBindings(
						roleBinding("reader", "Three", "User:jessie"),
					),
				},
				upsertParams{"One", "",
					*NewPolicy().SetRoleBindings(
						roleBinding("reader", "One", "User:jane"),
					),
				},
			},
			node: "Three",
			expected: *NewPolicy().SetRoleBindings(
				roleBinding("reader", "Three", "User:jane"),
				roleBinding("reader", "Three", "User:jessie"),
			),
		},
		{
			testName: "Partial tree (no node named 'one')",
			upserts: []upsertParams{
				upsertParams{"Two", "One",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:joe"),
					),
				},
				upsertParams{"Three", "Two",
					*NewPolicy().SetRoleBindings(
						rbactesting.RoleBinding("reader", "User:jessie"),
					),
				},
			},
			node: "Three",
			expected: *NewPolicy().SetRoleBindings(
				roleBinding("reader", "Three", "User:joe"),
				roleBinding("reader", "Three", "User:jessie"),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			tree := NewPolicyTree()
			for _, upsert := range tt.upserts {
				tree.Upsert(upsert.name, upsert.parent, false, upsert.policy)
			}
			result, err := tree.Eval(tt.node)
			if err != nil {
				t.Errorf("Eval returned an error: %v", err)
			}
			if !cmp.Equal(result, tt.expected, cmp.AllowUnexported(Policy{})) {
				t.Errorf("Mismatch: %v\nexpected:\n%v\nactual:\n%v",
					cmp.Diff(result, tt.expected, cmp.AllowUnexported(Policy{})),
					tt.expected, result)
			}
		})
	}
}
