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
	"reflect"
	"testing"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/clusterpolicycontroller"
	rbac_v1 "k8s.io/api/rbac/v1"
)

type UnpackTest struct {
	name      string
	resources []string
	module    clusterpolicycontroller.Module // populated in by UnpackTests
	field     string                         // populated in by UnpackTests
}

func (u *UnpackTest) Run(t *testing.T) {
	cp := &policyhierarchy_v1.ClusterPolicy{}
	cpsVal := reflect.ValueOf(&cp.Spec).Elem()
	listVal := cpsVal.FieldByName(u.field)
	newListVal := reflect.New(listVal.Type()).Elem()
	for _, name := range u.resources {
		inst := u.module.Instance()
		inst.SetName(name)
		newListVal = reflect.Append(newListVal, reflect.ValueOf(inst).Elem())
	}
	listVal.Set(newListVal)

	actual := u.module.Extract(cp)
	if len(actual) != len(u.resources) {
		t.Errorf("Different length lists, actual: %d expected: %d", len(actual), len(u.resources))
	}

	for i := 0; i < len(u.resources); i++ {
		if u.resources[i] != actual[i].GetName() {
			t.Errorf("Name mismatch, actual: %s expected: %s", actual[i].GetName(), u.resources[i])
		}
	}
}

type UnpackTests struct {
	module    clusterpolicycontroller.Module
	field     string
	testcases []UnpackTest
}

func (u *UnpackTests) Run(t *testing.T) {
	for _, testcase := range u.testcases {
		testcase.module = u.module
		testcase.field = u.field
		t.Run(testcase.name, testcase.Run)
	}
}

func TestRoleBindingsUnpack(t *testing.T) {
	testCases := UnpackTests{
		module: NewClusterRoleBindings(nil, nil),
		field:  "ClusterRoleBindingsV1",
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

func TestRoleBindingsEqual(t *testing.T) {
	clusterRolesModule := NewClusterRoleBindings(nil, nil)
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
