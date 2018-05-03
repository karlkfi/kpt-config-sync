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

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	test "github.com/google/nomos/pkg/syncer/policyhierarchycontroller/testing"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoles(t *testing.T) {
	admin := rbac_v1.Role{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "admin",
		},
		Rules: []rbac_v1.PolicyRule{
			rbac_v1.PolicyRule{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
		},
	}
	editor := rbac_v1.Role{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "editor",
		},
		Rules: []rbac_v1.PolicyRule{
			rbac_v1.PolicyRule{
				Verbs:     []string{"*"},
				APIGroups: []string{""},
				Resources: []string{"pods", "deployments", "services"},
			},
		},
	}
	bob := rbac_v1.Role{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "bob",
		},
		Rules: []rbac_v1.PolicyRule{
			rbac_v1.PolicyRule{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"pods"},
			},
		},
	}

	testSuite := test.ModuleTest{
		Module: NewRoleModule(nil, nil),
		Equals: test.ModuleEqualTestcases{
			test.ModuleEqualTestcase{
				Name:        "Both empty",
				LHS:         &rbac_v1.Role{},
				RHS:         &rbac_v1.Role{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name:        "Nil vs non nil lists",
				LHS:         &rbac_v1.Role{Rules: []rbac_v1.PolicyRule{}},
				RHS:         &rbac_v1.Role{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name:        "Different Rules",
				LHS:         &rbac_v1.Role{Rules: bob.Rules},
				RHS:         &rbac_v1.Role{Rules: admin.Rules},
				ExpectEqual: false,
			},
		},
		Aggregation: test.ModuleAggregationTestcases{
			test.ModuleAggregationTestcase{
				Name:       "Both empty",
				Aggregated: &AggregatedRole{},
				PolicyNode: &policyhierarchy_v1.PolicyNode{
					Spec: policyhierarchy_v1.PolicyNodeSpec{
						Type: policyhierarchy_v1.Namespace,
					},
				},
				Expect: hierarchy.Instances{},
			},
			test.ModuleAggregationTestcase{
				Name:       "Base case to workload namespace",
				Aggregated: &AggregatedRole{},
				PolicyNode: &policyhierarchy_v1.PolicyNode{
					Spec: policyhierarchy_v1.PolicyNodeSpec{
						Type:    policyhierarchy_v1.Namespace,
						RolesV1: []rbac_v1.Role{admin, editor},
					},
				},
				Expect: hierarchy.Instances{
					&admin,
					&editor,
				},
			},
			test.ModuleAggregationTestcase{
				Name:       "Workload namespace empty",
				Aggregated: &AggregatedRole{},
				PolicyNode: &policyhierarchy_v1.PolicyNode{
					Spec: policyhierarchy_v1.PolicyNodeSpec{
						Type: policyhierarchy_v1.Namespace,
					},
				},
				Expect: hierarchy.Instances{},
			},
			test.ModuleAggregationTestcase{
				Name:       "Ignore policyspaces",
				Aggregated: &AggregatedRole{},
				PolicyNode: &policyhierarchy_v1.PolicyNode{
					Spec: policyhierarchy_v1.PolicyNodeSpec{
						Type:    policyhierarchy_v1.Policyspace,
						RolesV1: []rbac_v1.Role{admin, editor},
					},
				},
				Expect: hierarchy.Instances{},
			},
		},
	}
	testSuite.RunAll(t)
}
