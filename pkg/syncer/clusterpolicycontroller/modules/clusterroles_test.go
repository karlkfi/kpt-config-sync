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
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRolesUnpack(t *testing.T) {
	testCases := UnpackTests{
		module: NewClusterRolesModule(nil, nil),
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
	clusterRolesModule := NewClusterRolesModule(nil, nil)
	testCases := []struct {
		name         string
		clusterRoleL *rbac_v1.ClusterRole
		clusterRoleR *rbac_v1.ClusterRole
		wantEqual    bool
	}{
		{
			name:      "nil roles",
			wantEqual: true,
		},
		{
			name: "different rules",
			clusterRoleL: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "admin",
				},
				Rules: []rbac_v1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			clusterRoleR: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "readonly",
				},
				Rules: []rbac_v1.PolicyRule{
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
			clusterRoleL: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbac_v1.AggregationRule{
					ClusterRoleSelectors: []meta_v1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-edit": "true",
							},
						},
					},
				},
			},
			clusterRoleR: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbac_v1.AggregationRule{
					ClusterRoleSelectors: []meta_v1.LabelSelector{
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
			clusterRoleL: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbac_v1.AggregationRule{
					ClusterRoleSelectors: []meta_v1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-edit": "true",
							},
						},
					},
				},
				Rules: []rbac_v1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			clusterRoleR: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbac_v1.AggregationRule{
					ClusterRoleSelectors: []meta_v1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbac_v1.PolicyRule{
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
			clusterRoleL: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbac_v1.AggregationRule{
					ClusterRoleSelectors: []meta_v1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbac_v1.PolicyRule{
					{
						Verbs:     []string{"get list"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			clusterRoleR: &rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "admin",
				},
				AggregationRule: &rbac_v1.AggregationRule{
					ClusterRoleSelectors: []meta_v1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"rbac.authorization.k8s.io/aggregate-to-view": "true",
							},
						},
					},
				},
				Rules: []rbac_v1.PolicyRule{
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
