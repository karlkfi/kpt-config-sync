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
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newNode(name string, parent string, policyspace bool) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: policyspace,
			Parent:      parent,
		},
	}
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
	v.Add(newNode("root", "", true))
	v.Add(newNode("child1", "root", true))
	v.Add(newNode("child2", "root", true))

	child1_1 := newNode("child1-1", "child1", false)
	v.Add(child1_1)

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
	v.Add(newNode("root", "", true))
	v.Add(newNode("child1", "root", true))
	v.Add(newNode("child2", "root", true))

	child1_1 := newNode("child1-1", "child1", false)
	v.Add(child1_1)

	if err := v.Validate(); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}

	child1_1.Spec.Parent = "child2"
	if err := v.Remove(child1_1); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}
}

func TestMultipleRoots(t *testing.T) {
	v := New()
	v.Add(newNode("root", "", true))
	v.Add(newNode("child1", "root", true))
	v.Add(newNode("child2", "child1", false))

	if err := v.checkRoots(); err != nil {
		t.Errorf("Multiple roots state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Multiple roots state should be OK %s %s", err, spew.Sdump(v))
	}

	v.Add(newNode("root2", "", true))
	v.Add(newNode("child2-1", "root2", false))
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

	v.Add(newNode("root", "", true))
	v.Add(newNode("child1", "root", true))
	v.Add(newNode("child2", "child1", false))
	v.Add(newNode("root2", "", true))
	v.Add(newNode("child2-1", "root2", false))

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

	v.Add(newNode("child1", "root", true))
	v.Add(newNode("child2", "child1", false))

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
	v.Add(root)
	v.Add(child1)
	v.Add(child2)

	if err := v.checkWorkingNamespace(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}

	child1.Spec.Policyspace = false
	if err := v.checkWorkingNamespace(); err == nil {
		t.Errorf("Should have detected intermediate node working namespace error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected intermediate node working namespace error")
	}
}

func TestRootWorkingNamespace(t *testing.T) {
	v := New()
	root := newNode("root", "", true)
	v.Add(root)
	if err := v.checkWorkingNamespace(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}

	root.Spec.Policyspace = false
	if err := v.checkWorkingNamespace(); err == nil {
		t.Errorf("Should have detected leaf node working namespace error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected leaf node working namespace error")
	}
}

func TestCycle(t *testing.T) {
	v := New()
	v.Add(newNode("root", "", true))
	v.Add(newNode("child1", "root", true))
	v.Add(newNode("child2", "child1", false))

	if err := v.checkCycles(); err != nil {
		t.Errorf("No cycle should exist %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("No cycle should exist %s %s", err, spew.Sdump(v))
	}

	v.Add(newNode("child3", "child4", true))
	v.Add(newNode("child4", "child3", true))
	if err := v.checkCycles(); err == nil {
		t.Errorf("Should have detected cycle")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected cycle")
	}
}

func TestPolicySpaceWithRoles(t *testing.T) {
	v := New()

	v.Add(newNode("root", "", true))

	policySpaceWithRole := newNode("policyspacewithrole", "root", true)
	policySpaceWithRole.Spec.Policies.RolesV1 = []rbac_v1.Role{{}}
	v.Add(policySpaceWithRole)

	v.Add(newNode("child2", "policyspacewithrole", false))

	if err := v.checkPolicySpaceRoles(); err == nil {
		t.Errorf("Should have detected policy space roles error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected policy space roles multiple roots error")
	}
}

func TestAddOrphan(t *testing.T) {
	v := New()
	v.Add(newNode("root", "", true))
	v.Add(newNode("child1", "root", true))
	v.Add(newNode("child2", "child1", false))

	if err := v.checkParents(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}

	v.Add(newNode("orphan", "nonexistantparent", false))

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

func TestRemovePolicySpace(t *testing.T) {
	v := New()
	v.Add(newNode("root", "", true))

	policySpace := newNode("child1", "root", true)
	v.Add(policySpace)

	v.Add(newNode("child2", "child1", false))
	v.Add(newNode("child3", "child1", false))

	if err := v.checkParents(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should not have detected missing parent error: %v", err)
	}

	if err := v.Remove(policySpace); err == nil {
		t.Errorf("Should have detected parent delete error")
	}
}
