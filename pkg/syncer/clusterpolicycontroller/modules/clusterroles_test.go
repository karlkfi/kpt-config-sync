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
package modules

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRolesUnpack(t *testing.T) {
	testCases := UnpackTests{
		module: NewClusterRoles(nil, nil),
		field:  "ClusterRolesV1",
		testcases: []UnpackTest{
			UnpackTest{
				name:      "none",
				resources: []string{},
			},
			UnpackTest{
				name:      "one",
				resources: []string{"foo"},
			},
			UnpackTest{
				name:      "two",
				resources: []string{"foo", "bar"},
			},
			UnpackTest{
				name:      "three",
				resources: []string{"foo", "bar", "baz"},
			},
		},
	}
	testCases.Run(t)
}

func TestRolesEqual(t *testing.T) {
	clusterRolesModule := NewClusterRoles(nil, nil)
	testCases := []struct {
		name         string
		clusterRoleL *rbacv1.ClusterRole
		clusterRoleR *rbacv1.ClusterRole
		wantEqual    bool
	}{
		{
			name:      "nil roles",
			wantEqual: true,
		},
		{
			name: "different rules",
			clusterRoleL: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			clusterRoleR: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "readonly",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "different aggregation rules",
			clusterRoleL: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-edit": "true",
							},
						},
					},
				},
			},
			clusterRoleR: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "different aggregation rules, same rules",
			clusterRoleL: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-edit": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			clusterRoleR: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "same aggregation rules, same rules",
			clusterRoleL: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			clusterRoleR: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbacv1.AggregationRule{
					ClusterRoleSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			wantEqual: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := clusterRolesModule.Equal(tt.clusterRoleL, tt.clusterRoleR)
			if got != tt.wantEqual {
				t.Errorf("expected %v got %v", tt.wantEqual, got)
			}
		})
	}
}
