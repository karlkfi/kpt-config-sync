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

	"github.com/google/nomos/pkg/syncer/hierarchy"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	test "github.com/google/nomos/pkg/syncer/policyhierarchycontroller/testing"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RenameRoleBinding(r *rbac_v1.RoleBinding, name string) *rbac_v1.RoleBinding {
	c := r.DeepCopy()
	c.Name = name
	return c
}

func WithRoleBindings(
	name string,
	t policyhierarchy_v1.PolicyNodeType,
	rbs ...rbac_v1.RoleBinding) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Type:           t,
			RoleBindingsV1: rbs,
		},
	}
}

func TestRoleBindings(t *testing.T) {
	admins := &rbac_v1.RoleBinding{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "admins",
		},
		Subjects: []rbac_v1.Subject{
			rbac_v1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "alice@megacorp.org",
			},
		},
		RoleRef: rbac_v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "admin",
		},
	}
	editors := &rbac_v1.RoleBinding{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "editors",
		},
		Subjects: []rbac_v1.Subject{
			rbac_v1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "edward@megacorp.org",
			},
		},
		RoleRef: rbac_v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "editor",
		},
	}
	bobs := &rbac_v1.RoleBinding{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "bobs",
		},
		Subjects: []rbac_v1.Subject{
			rbac_v1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "bob@megacorp.org",
			},
		},
		RoleRef: rbac_v1.RoleRef{
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
				LHS:         &rbac_v1.RoleBinding{},
				RHS:         &rbac_v1.RoleBinding{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Nil vs non nil lists",
				LHS: &rbac_v1.RoleBinding{
					Subjects: []rbac_v1.Subject{},
					RoleRef:  rbac_v1.RoleRef{},
				},
				RHS:         &rbac_v1.RoleBinding{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Different RoleRef",
				LHS: &rbac_v1.RoleBinding{
					Subjects: []rbac_v1.Subject{},
					RoleRef:  admins.RoleRef,
				},
				RHS: &rbac_v1.RoleBinding{
					Subjects: []rbac_v1.Subject{},
					RoleRef:  bobs.RoleRef,
				},
				ExpectEqual: false,
			},
			test.ModuleEqualTestcase{
				Name: "Different Subjects",
				LHS: &rbac_v1.RoleBinding{
					Subjects: bobs.Subjects,
					RoleRef:  rbac_v1.RoleRef{},
				},
				RHS: &rbac_v1.RoleBinding{
					Subjects: admins.Subjects,
					RoleRef:  rbac_v1.RoleRef{},
				},
				ExpectEqual: false,
			},
		},
		Aggregation: test.ModuleAggregationTestcases{
			test.ModuleAggregationTestcase{
				Name: "Both empty",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					&policyhierarchy_v1.PolicyNode{},
				},
				Expect: hierarchy.Instances{},
			},
			test.ModuleAggregationTestcase{
				Name: "Base case",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithRoleBindings("current", policyhierarchy_v1.Namespace, *admins),
				},
				Expect: hierarchy.Instances{
					RenameRoleBinding(admins, "admins"),
				},
			},
			test.ModuleAggregationTestcase{
				Name: "Node empty",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithRoleBindings("editors", policyhierarchy_v1.Policyspace, *editors),
					WithRoleBindings("empty", policyhierarchy_v1.Namespace),
				},
				Expect: hierarchy.Instances{
					RenameRoleBinding(editors, "editors.editors"),
				},
			},
			test.ModuleAggregationTestcase{
				Name: "Base case",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithRoleBindings("base", policyhierarchy_v1.Namespace, *admins),
				},
				Expect: hierarchy.Instances{
					RenameRoleBinding(admins, "admins"),
				},
			},
			test.ModuleAggregationTestcase{
				Name: "Aggregation case",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithRoleBindings("parent", policyhierarchy_v1.Policyspace, *bobs, *editors),
					WithRoleBindings("current", policyhierarchy_v1.Namespace, *admins),
				},
				Expect: hierarchy.Instances{
					RenameRoleBinding(admins, "admins"),
					RenameRoleBinding(bobs, "parent.bobs"),
					RenameRoleBinding(editors, "parent.editors"),
				},
			},
		},
	}
	testSuite.RunAll(t)
}
