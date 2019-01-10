/*
Copyright 2017 The Nomos Authors.
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

package validator

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("got error: %s", err)
	}
}

func newNode(name string, parent string, policyspace bool) *v1.PolicyNode {
	pnt := v1.Namespace
	if policyspace {
		pnt = v1.Policyspace
	}
	return &v1.PolicyNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PolicyNodeSpec{
			Type:   pnt,
			Parent: parent,
		},
	}
}

func setResources(pn *v1.PolicyNode, roleNames, roleBindingNames []string) {
	var roles []rbacv1.Role
	for _, rn := range roleNames {
		role := rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name: rn,
			},
		}
		roles = append(roles, role)
	}

	var roleBindings []rbacv1.RoleBinding
	for _, rn := range roleBindingNames {
		roleBinding := rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: rn,
			},
		}
		roleBindings = append(roleBindings, roleBinding)
	}

	pn.Spec.RolesV1 = roles
	pn.Spec.RoleBindingsV1 = roleBindings
}

func TestDuplicateName(t *testing.T) {
	v := New()
	if err := v.Add(newNode("root", "", true)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child1", "root", true)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child2", "child1", false)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child1-1", "root", true)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child2", "child1-1", false)); err == nil {
		t.Errorf("Add duplicate node should have encountered error")
	}
	if err := v.Add(newNode("", "child1-1", false)); err == nil {
		t.Errorf("Add unnamed node should have encountered error")
	}
}

func TestMove(t *testing.T) {
	v := New()
	checkErr(t, v.Add(newNode("root", "", true)))
	checkErr(t, v.Add(newNode("child1", "root", true)))
	checkErr(t, v.Add(newNode("child2", "root", true)))

	child1_1 := newNode("child1-1", "child1", false)
	checkErr(t, v.Add(child1_1))

	if err := v.Validate(); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}

	child1_1.Spec.Parent = "child2"
	if err := v.Update(child1_1); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}
	if v.policyNodes["child1-1"].Spec.Parent != "child2" {
		t.Errorf("Wrong parent for child")
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}
}

func TestRemove(t *testing.T) {
	v := New()
	checkErr(t, v.Add(newNode("root", "", true)))
	checkErr(t, v.Add(newNode("child1", "root", true)))
	checkErr(t, v.Add(newNode("child2", "root", true)))

	child1_1 := newNode("child1-1", "child1", false)
	checkErr(t, v.Add(child1_1))

	if err := v.Validate(); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}

	child1_1.Spec.Parent = "child2"
	if err := v.Remove(child1_1); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}
}

func TestRootCannotBeNamespace(t *testing.T) {
	v := New()
	checkErr(t, v.Add(newNode("root", "", false)))
	if err := v.checkRoots(); err == nil {
		t.Errorf("Should have namespace is root error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have namespace is root error")
	}
}

func TestMultipleRoots(t *testing.T) {
	v := New()
	checkErr(t, v.Add(newNode("root", "", true)))
	checkErr(t, v.Add(newNode("child1", "root", true)))
	checkErr(t, v.Add(newNode("child2", "child1", false)))

	if err := v.checkRoots(); err != nil {
		t.Errorf("Multiple roots state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Multiple roots state should be OK %s %s", err, spew.Sdump(v))
	}

	checkErr(t, v.Add(newNode("root2", "", true)))
	checkErr(t, v.Add(newNode("child2-1", "root2", false)))
	if err := v.checkRoots(); err == nil {
		t.Errorf("Should have detected multiple roots error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected multiple roots error")
	}
}

func TestAllowMultipleRoots(t *testing.T) {
	v := New()
	v.AllowMultipleRoots = true

	checkErr(t, v.Add(newNode("root", "", true)))
	checkErr(t, v.Add(newNode("child1", "root", true)))
	checkErr(t, v.Add(newNode("child2", "child1", false)))
	checkErr(t, v.Add(newNode("root2", "", true)))
	checkErr(t, v.Add(newNode("child2-1", "root2", false)))

	if err := v.checkRoots(); err != nil {
		t.Errorf("Should not have detected multiple roots error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected multiple roots error: %v", err)
	}
}

func TestNoRoot(t *testing.T) {
	v := New()
	v.AllowMultipleRoots = true

	checkErr(t, v.Add(newNode("child1", "root", true)))
	checkErr(t, v.Add(newNode("child2", "child1", false)))

	if err := v.checkRoots(); err == nil {
		t.Errorf("Should have detected no roots error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected no roots error")
	}
}

func TestWorkingNamespace(t *testing.T) {
	v := New()
	root := newNode("root", "", true)
	child1 := newNode("child1", "root", true)
	child2 := newNode("child2", "child1", false)
	checkErr(t, v.Add(root))
	checkErr(t, v.Add(child1))
	checkErr(t, v.Add(child2))

	if err := v.Validate(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}

	child2.Spec.Type = v1.Policyspace
	v = New()
	checkErr(t, v.Add(root))
	checkErr(t, v.Add(child1))
	checkErr(t, v.Add(child2))
	if err := v.Validate(); err != nil {
		t.Errorf("Should not error for policyspace leaf node: %v", err)
	}
}

func TestRootWorkingNamespace(t *testing.T) {
	v := New()
	root := newNode("root", "", true)
	checkErr(t, v.Add(root))
	if err := v.checkRoots(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}

	v = New()
	root.Spec.Type = v1.Namespace
	checkErr(t, v.Add(root))
	if err := v.checkRoots(); err == nil {
		t.Errorf("Should have detected leaf node working namespace error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected leaf node working namespace error")
	}
}

func TestCycle(t *testing.T) {
	v := New()
	checkErr(t, v.Add(newNode("root", "", true)))
	checkErr(t, v.Add(newNode("child1", "root", true)))
	checkErr(t, v.Add(newNode("child2", "child1", false)))

	if err := v.checkCycles(); err != nil {
		t.Errorf("No cycle should exist %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("No cycle should exist %s %s", err, spew.Sdump(v))
	}

	checkErr(t, v.Add(newNode("child3", "child4", true)))
	checkErr(t, v.Add(newNode("child4", "child3", true)))
	if err := v.checkCycles(); err == nil {
		t.Errorf("Should have detected cycle")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected cycle")
	}
}

func TestPolicySpaceWithRoles(t *testing.T) {
	v := New()

	checkErr(t, v.Add(newNode("root", "", true)))

	policySpaceWithRole := newNode("policyspacewithrole", "root", true)
	policySpaceWithRole.Spec.RolesV1 = []rbacv1.Role{{}}
	checkErr(t, v.Add(policySpaceWithRole))

	checkErr(t, v.Add(newNode("child2", "policyspacewithrole", false)))

	if err := v.checkPolicySpaceRoles(); err == nil {
		t.Errorf("Should have detected policy space roles error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected policy space roles multiple roots error")
	}
}

func TestAddOrphan(t *testing.T) {
	v := New()
	checkErr(t, v.Add(newNode("root", "", true)))
	checkErr(t, v.Add(newNode("child1", "root", true)))
	checkErr(t, v.Add(newNode("child2", "child1", false)))

	if err := v.checkParents(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}

	checkErr(t, v.Add(newNode("orphan", "nonexistantparent", false)))

	if err := v.checkParents(); err == nil {
		t.Errorf("Should have detected missing parent error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected missing parent error")
	}

	v.AllowOrphanAdds = true
	if err := v.checkParents(); err != nil {
		t.Errorf("Should have bypassed missing parent error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should have bypassed missing parent error: %v", err)
	}
}

func TestRemoveParents(t *testing.T) {
	v := New()
	root := newNode("root", "", true)
	checkErr(t, v.Add(root))

	policySpace := newNode("child1", "root", true)
	checkErr(t, v.Add(policySpace))

	checkErr(t, v.Add(newNode("child2", "child1", false)))
	checkErr(t, v.Add(newNode("child3", "child1", false)))

	if err := v.checkParents(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}

	if err := v.Remove(policySpace); err == nil {
		t.Errorf("Should have detected policyspace delete error")
	}
	if err := v.Remove(root); err == nil {
		t.Errorf("Should have detected root delete error")
	}
}

func TestRemoveRemainingRoot(t *testing.T) {
	nodes := []*v1.PolicyNode{
		newNode("root", "", true),
		newNode("child1", "root", true),
		newNode("child2", "child1", false),
		newNode("child3", "child1", false),
	}

	v := New()
	for _, node := range nodes {
		if err := v.Add(node); err != nil {
			t.Errorf("Should not have errored when adding node %s: %v", node.Name, err)
		}
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected validation error after adding nodes: %v", err)
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if err := v.Remove(node); err != nil {
			t.Errorf("Should not have errored when removing node %s: %v", node.Name, err)
		}
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected validation error after deleting nodes: %v", err)
	}
}

func TestDuplicateResourcesInNode(t *testing.T) {
	root := newNode("root", "", true)
	child := newNode("child", "root", false)
	setResources(child, []string{"role", "otherrole"}, []string{"rolebinding", "otherrolebinding"})

	v := New()
	for _, pn := range []*v1.PolicyNode{root, child} {
		if err := v.Add(pn); err != nil {
			t.Errorf("Should not have errored when adding %q node: %v", pn.Name, err)
		}
	}

	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected duplicate names error: %v", err)
	}

	checkErr(t, v.Remove(child))
	setResources(child, []string{"role", "role"}, []string{"rolebinding"})
	checkErr(t, v.Add(child))
	if err := v.Validate(); err == nil {
		t.Error("Should have detected duplicate roles error")
	}

	checkErr(t, v.Remove(child))
	setResources(child, []string{"role"}, []string{"rolebinding", "rolebinding"})
	checkErr(t, v.Add(child))
	if err := v.Validate(); err == nil {
		t.Error("Should have detected duplicate rolebindings error")
	}
}
