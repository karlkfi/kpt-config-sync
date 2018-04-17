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
package modules

import (
	"testing"

	rbac_v1 "k8s.io/api/rbac/v1"
)

func TestRoleBindingsEqual(t *testing.T) {
	clusterRolesModule := NewClusterRoleBindingsModule(nil, nil)
	testCases := []struct {
		name         string
		roleBindingL *rbac_v1.ClusterRoleBinding
		roleBindingR *rbac_v1.ClusterRoleBinding
		wantEqual    bool
	}{
		{
			name:      "nil rolebindings",
			wantEqual: true,
		},
		{
			name:         "empty rolebindings",
			roleBindingL: &rbac_v1.ClusterRoleBinding{},
			roleBindingR: &rbac_v1.ClusterRoleBinding{},
			wantEqual:    true,
		},
		{
			name: "nil vs empty subjects",
			roleBindingL: &rbac_v1.ClusterRoleBinding{
				Subjects: []rbac_v1.Subject{},
			},
			roleBindingR: &rbac_v1.ClusterRoleBinding{},
			wantEqual:    true,
		},
		{
			name: "different RoleRefs",
			roleBindingL: &rbac_v1.ClusterRoleBinding{
				Subjects: []rbac_v1.Subject{},
				RoleRef: rbac_v1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "admin",
				},
			},
			roleBindingR: &rbac_v1.ClusterRoleBinding{
				Subjects: []rbac_v1.Subject{},
				RoleRef: rbac_v1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "readonly",
				},
			},
			wantEqual: false,
		},
		{
			name: "different subjects",
			roleBindingL: &rbac_v1.ClusterRoleBinding{
				Subjects: []rbac_v1.Subject{
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "User",
						Name:     "alice@megacorp.org",
					},
				},
				RoleRef: rbac_v1.RoleRef{},
			},
			roleBindingR: &rbac_v1.ClusterRoleBinding{
				Subjects: []rbac_v1.Subject{
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "User",
						Name:     "bob@megacorp.org",
					},
				},
				RoleRef: rbac_v1.RoleRef{},
			},
			wantEqual: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := clusterRolesModule.Equal(tt.roleBindingL, tt.roleBindingR)
			if got != tt.wantEqual {
				t.Errorf("expected %v got %v", tt.wantEqual, got)
			}
		})
	}
}
