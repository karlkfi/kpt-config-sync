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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleBindings(t *testing.T) {
	admins := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admins",
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "alice@megacorp.org",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "admin",
		},
	}
	bobs := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bobs",
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "bob@megacorp.org",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "bob",
		},
	}

	testSuite := test.ModuleTest{
		Module: NewRoleBinding(nil, nil),
		Equals: test.ModuleEqualTestcases{
			test.ModuleEqualTestcase{
				Name:        "Both empty",
				LHS:         &rbacv1.RoleBinding{},
				RHS:         &rbacv1.RoleBinding{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Nil vs non nil lists",
				LHS: &rbacv1.RoleBinding{
					Subjects: []rbacv1.Subject{},
					RoleRef:  rbacv1.RoleRef{},
				},
				RHS:         &rbacv1.RoleBinding{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Different RoleRef",
				LHS: &rbacv1.RoleBinding{
					Subjects: []rbacv1.Subject{},
					RoleRef:  admins.RoleRef,
				},
				RHS: &rbacv1.RoleBinding{
					Subjects: []rbacv1.Subject{},
					RoleRef:  bobs.RoleRef,
				},
				ExpectEqual: false,
			},
			test.ModuleEqualTestcase{
				Name: "Different Subjects",
				LHS: &rbacv1.RoleBinding{
					Subjects: bobs.Subjects,
					RoleRef:  rbacv1.RoleRef{},
				},
				RHS: &rbacv1.RoleBinding{
					Subjects: admins.Subjects,
					RoleRef:  rbacv1.RoleRef{},
				},
				ExpectEqual: false,
			},
		},
	}
	testSuite.RunAll(t)
}
