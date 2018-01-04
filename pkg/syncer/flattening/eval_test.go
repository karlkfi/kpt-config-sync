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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/stolos/pkg/testing/rbactesting"
	rbac "k8s.io/api/rbac/v1"
)

func TestFlattenRoleBindingsFunc(t *testing.T) {
	tests := []struct {
		name          string
		parent, child Policy
		expected      Policy
	}{
		{
			name: "Hierarchical RoleBinding merge",
			parent: *NewPolicy().SetRoleBindings(
				rbactesting.RoleBinding("reader", "User:joe"),
			),
			child: *NewPolicy().SetRoleBindings(
				rbactesting.RoleBinding("reader", "User:jessie"),
			),
			expected: *NewPolicy().SetRoleBindings(
				rbactesting.RoleBinding("reader", "User:joe"),
				rbactesting.RoleBinding("reader", "User:jessie")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := flatten("", tt.parent, tt.child)
			if !cmp.Equal(tt.expected, actual, cmp.AllowUnexported(Policy{})) {
				t.Errorf("Mismatch: %v",
					cmp.Diff(tt.expected, actual, cmp.AllowUnexported(Policy{})))
			}
		})
	}
}

func TestFlattenRoles(t *testing.T) {
	tests := []struct {
		name          string
		parent, child Policy
		expected      Policy
	}{
		{
			name: "Hierarchical Role merge without conflict",
			parent: *NewPolicy().SetRoles(
				rbactesting.Role("user1", []rbac.PolicyRule{}),
			),
			child: *NewPolicy().SetRoles(
				rbactesting.Role("user2", []rbac.PolicyRule{}),
			),
			expected: *NewPolicy().SetRoles(
				rbactesting.Role("user1", []rbac.PolicyRule{}),
				rbactesting.Role("user2", []rbac.PolicyRule{}),
			),
		},
		{
			name: "Hierarchical Role merge with conflict",
			parent: *NewPolicy().SetRoles(
				rbactesting.Role(
					"user0",
					[]rbac.PolicyRule{
						rbactesting.PolicyRule(
							[]string{"*"},
							[]string{"get"},
							[]string{"orgs"})}),
				rbactesting.Role("user1", []rbac.PolicyRule{}),
			),
			child: *NewPolicy().SetRoles(
				rbactesting.Role(
					"user0",
					[]rbac.PolicyRule{
						rbactesting.PolicyRule(
							[]string{"*"},
							[]string{"get"},
							[]string{"ous"})}),
				rbactesting.Role("user2", []rbac.PolicyRule{}),
			),
			expected: *NewPolicy().SetRoles(
				// The role definition from parent wins.
				rbactesting.Role(
					"user0",
					[]rbac.PolicyRule{
						rbactesting.PolicyRule(
							[]string{"*"},
							[]string{"get"},
							[]string{"orgs"})}),
				rbactesting.Role("user1", []rbac.PolicyRule{}),
				rbactesting.Role("user2", []rbac.PolicyRule{}),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := flatten("", tt.parent, tt.child)
			if !cmp.Equal(tt.expected, actual, cmp.AllowUnexported(Policy{})) {
				t.Errorf("Mismatch: %v",
					cmp.Diff(tt.expected, actual, cmp.AllowUnexported(Policy{})))
			}
		})
	}

}
