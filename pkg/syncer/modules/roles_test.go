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
		Module: NewRole(nil, nil),
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
	}
	testSuite.RunAll(t)
}
