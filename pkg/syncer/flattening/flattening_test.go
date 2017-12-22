package flattening

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/stolos/pkg/testing/rbactesting"
	rbac "k8s.io/api/rbac/v1"
)

func TestPolicyNode(t *testing.T) {
	tree := NewPolicyTree()
	tree.Upsert(Name("Node"), NoParent, []struct{}{})
	if tree.Lookup(Name("Node")) == nil {
		t.Errorf("Node not found, but should be.")
	}
}

func TestFlattenRoleBindingsFunc(t *testing.T) {
	tests := []struct {
		name          string
		parent, child []rbac.RoleBinding
		expected      []rbac.RoleBinding
	}{
		{
			name: "Hierarchical RoleBinding merge",
			parent: []rbac.RoleBinding{
				rbactesting.RoleBinding("reader", "User:joe"),
			},
			child: []rbac.RoleBinding{
				rbactesting.RoleBinding("reader", "User:jessie"),
			},
			expected: []rbac.RoleBinding{
				rbactesting.RoleBinding("reader", "User:joe"),
				rbactesting.RoleBinding("reader", "User:jessie"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FlattenRoleBindings(&tt.parent, &tt.child).(*[]rbac.RoleBinding)
			if !cmp.Equal(&tt.expected, actual) {
				t.Errorf("Mismatch: %v", cmp.Diff(&tt.expected, actual))
			}

		})
	}
}

// role is a quick role definition for tests where most of the role content
// does not mattter.
func role(user string, verb string) rbac.Role {
	return rbactesting.Role(
		user,
		[]rbac.PolicyRule{
			rbactesting.PolicyRule(
				[]string{"*"}, []string{verb}, []string{}),
		})
}

func TestFlattenRolesFunc(t *testing.T) {
	tests := []struct {
		name          string
		parent, child []rbac.Role
		expected      []rbac.Role
	}{
		{
			name: "Hierarchical conflicting role merge higer up wins",
			parent: []rbac.Role{
				role("User:joe", "get"),
			},
			child: []rbac.Role{
				role("User:joe", "list"),
			},
			expected: []rbac.Role{
				role("User:joe", "get"),
			},
		},
		{
			name: "Hierarchical multiple role merge",
			parent: []rbac.Role{
				role("User:joe", "get"),
				role("User:jessie", "get"),
			},
			child: []rbac.Role{
				role("User:joe", "list"),
				role("User:jane", "get"),
			},
			expected: []rbac.Role{
				role("User:joe", "get"),
				role("User:jessie", "get"),
				role("User:jane", "get"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FlattenRoles(&tt.parent, &tt.child).(*[]rbac.Role)
			if !cmp.Equal(&tt.expected, actual) {
				t.Errorf("Mismatch: %v", cmp.Diff(&tt.expected, actual))
			}

		})
	}
}
func TestFlatten(t *testing.T) {
	type upsertParams struct {
		name   Name
		parent Parent
		policy Policy
	}
	tests := []struct {
		testName string
		// These are the upsert commands that will be executed.
		upserts []upsertParams
		// Choose the flattening behavior to use.  Roles are squashed,
		// RoleBindings are appended, for example.
		flattener FlattenPolicyFunc
		// The name of the node to look up at the end.
		node Name
		// The expected resulting policy for the said node.
		expected Policy
	}{
		{
			testName: "nil values",
			upserts: []upsertParams{
				upsertParams{Name("One"), NoParent, &[]rbac.RoleBinding{}},
				upsertParams{Name("Two"), Parent("One"), &[]rbac.RoleBinding{}},
				upsertParams{Name("Three"), Parent("Two"), &[]rbac.RoleBinding{}},
			},
			node:      "Three",
			expected:  &[]rbac.RoleBinding{},
			flattener: FlattenRoleBindings,
		},
		{
			testName: "Upsert RoleBindings in order.",
			upserts: []upsertParams{
				upsertParams{
					Name("One"), NoParent,
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jane"),
					},
				},
				upsertParams{Name("Two"), Parent("One"),
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:joe"),
					},
				},
				upsertParams{Name("Three"), Parent("Two"),
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jessie"),
					},
				},
			},
			node: "Three",
			expected: &[]rbac.RoleBinding{
				rbactesting.RoleBinding("reader", "User:jane"),
				rbactesting.RoleBinding("reader", "User:joe"),
				rbactesting.RoleBinding("reader", "User:jessie"),
			},
			flattener: FlattenRoleBindings,
		},
		{
			testName: "Basic Upsert RoleBindings out of order",
			upserts: []upsertParams{
				upsertParams{Name("Two"), Parent("One"),
					&[]rbac.RoleBinding{rbac.RoleBinding{}},
				},
				upsertParams{
					Name("One"), NoParent,
					&[]rbac.RoleBinding{rbac.RoleBinding{}},
				},
			},
			node: "Two",
			expected: &[]rbac.RoleBinding{
				rbac.RoleBinding{},
				rbac.RoleBinding{},
			},
			flattener: FlattenRoleBindings,
		},
		{
			testName: "Upsert out of order",
			upserts: []upsertParams{
				upsertParams{Name("Three"), Parent("Two"),
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jessie"),
					},
				},
				upsertParams{Name("Two"), Parent("One"),
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:joe"),
					},
				},
				upsertParams{
					Name("One"), NoParent,
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jane"),
					},
				},
			},
			node: "Three",
			expected: &[]rbac.RoleBinding{
				rbactesting.RoleBinding("reader", "User:jane"),
				rbactesting.RoleBinding("reader", "User:joe"),
				rbactesting.RoleBinding("reader", "User:jessie"),
			},
			flattener: FlattenRoleBindings,
		},
		{
			testName: "Reparenting",
			upserts: []upsertParams{
				upsertParams{Name("Three"), Parent("Two"),
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jessie"),
					},
				},
				upsertParams{Name("Two"), Parent("One"),
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:joe"),
					},
				},
				// "Three" is parented away from "Two" and onto "One".
				upsertParams{Name("Three"), Parent("One"),
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jessie"),
					},
				},
				upsertParams{
					Name("One"), NoParent,
					&[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jane"),
					},
				},
			},
			node: "Three",
			expected: &[]rbac.RoleBinding{
				rbactesting.RoleBinding("reader", "User:jane"),
				rbactesting.RoleBinding("reader", "User:jessie"),
			},
			flattener: FlattenRoleBindings,
		},
		{
			testName: "Flattening roles.",
			upserts: []upsertParams{
				upsertParams{Name("Three"), Parent("Two"),
					&[]rbac.Role{
						role("User:jessie", "get"),
					},
				},
				upsertParams{Name("Two"), Parent("One"),
					&[]rbac.Role{
						role("User:joe", "get"),
					},
				},
				upsertParams{
					Name("One"), NoParent,
					&[]rbac.Role{
						role("User:jane", "get"),
					},
				},
			},
			node: "Three",
			expected: &[]rbac.Role{
				role("User:jane", "get"),
				role("User:joe", "get"),
				role("User:jessie", "get"),
			},
			flattener: FlattenRoles,
		},
		{
			testName: "Flattening matching roles and higher up role wins",
			upserts: []upsertParams{
				upsertParams{Name("Two"), Parent("One"),
					&[]rbac.Role{
						role("User:joe", "get"),
					},
				},
				upsertParams{
					Name("One"), NoParent,
					&[]rbac.Role{
						role("User:joe", "list"),
					},
				},
			},
			node: "Two",
			expected: &[]rbac.Role{
				role("User:joe", "list"),
			},
			flattener: FlattenRoles,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			tree := NewPolicyTree()
			for _, upsert := range tt.upserts {
				tree.Upsert(upsert.name, upsert.parent, upsert.policy)
			}
			node := tree.Lookup(tt.node)
			if node == nil {
				t.Errorf("Node not found.")
				return
			}
			result := node.Eval(tt.flattener)
			if !cmp.Equal(result, tt.expected) {
				t.Errorf("Mismatch: %v", cmp.Diff(result, tt.expected))
			}
		})
	}
}
