/*
Copyright 2017 The Kubernetes Authors.
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
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newNode(name string, parent string, workingNamespace bool) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			WorkingNamespace: workingNamespace,
			Parent:           parent,
		},
	}
}

func TestDuplicateName(t *testing.T) {
	v := New()
	if err := v.Add(newNode("root", "", false)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child1", "root", false)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child2", "child1", true)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child1-1", "root", false)); err != nil {
		t.Errorf("Add should have been ok")
	}
	if err := v.Add(newNode("child2", "child1-1", true)); err == nil {
		t.Errorf("Add duplicate node should have encountered error")
	}
	if err := v.Add(newNode("", "child1-1", true)); err == nil {
		t.Errorf("Add unnamed node should have encountered error")
	}
}

func TestMove(t *testing.T) {
	v := New()
	v.Add(newNode("root", "", false))
	v.Add(newNode("child1", "root", false))
	v.Add(newNode("child2", "root", false))

	child1_1 := newNode("child1-1", "child1", true)
	v.Add(child1_1)

	if err := v.Validate(); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}

	child1_1.Spec.Parent = "child2"
	if err := v.Update(child1_1); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}
	if v.parents["child1-1"] != "child2" {
		t.Errorf("Wrong parent for child")
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Should be ok %s %s", err, spew.Sdump(v))
	}
}

func TestRemove(t *testing.T) {
	v := New()
	v.Add(newNode("root", "", false))
	v.Add(newNode("child1", "root", false))
	v.Add(newNode("child2", "root", false))

	child1_1 := newNode("child1-1", "child1", true)
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
	v.Add(newNode("root", "", false))
	v.Add(newNode("child1", "root", false))
	v.Add(newNode("child2", "child1", true))

	if err := v.checkMultipleRoots(); err != nil {
		t.Errorf("Multiple roots state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Multiple roots state should be OK %s %s", err, spew.Sdump(v))
	}

	v.Add(newNode("root2", "", false))
	v.Add(newNode("child2-1", "root2", true))
	if err := v.checkMultipleRoots(); err == nil {
		t.Errorf("Should have detected multiple roots error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected multiple roots error")
	}
}

func TestWorkingNamespace(t *testing.T) {
	v := New()
	root := newNode("root", "", false)
	child1 := newNode("child1", "root", false)
	child2 := newNode("child2", "child1", true)
	v.Add(root)
	v.Add(child1)
	v.Add(child2)

	if err := v.checkWorkingNamespace(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}

	child1.Spec.WorkingNamespace = true
	if err := v.checkWorkingNamespace(); err == nil {
		t.Errorf("Should have detected intermediate node working namespace error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected intermediate node working namespace error")
	}
}

func TestRootWorkingNamespace(t *testing.T) {
	v := New()
	root := newNode("root", "", false)
	v.Add(root)
	if err := v.checkWorkingNamespace(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("Working namespace state should be OK %s %s", err, spew.Sdump(v))
	}

	root.Spec.WorkingNamespace = true
	if err := v.checkWorkingNamespace(); err == nil {
		t.Errorf("Should have detected leaf node working namespace error")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected leaf node working namespace error")
	}
}

func TestCycle(t *testing.T) {
	v := New()
	v.Add(newNode("root", "", false))
	v.Add(newNode("child1", "root", false))
	v.Add(newNode("child2", "child1", true))

	if err := v.checkCycles(); err != nil {
		t.Errorf("No cycle should exist %s %s", err, spew.Sdump(v))
	}
	if err := v.Validate(); err != nil {
		t.Errorf("No cycle should exist %s %s", err, spew.Sdump(v))
	}

	v.Add(newNode("child3", "child4", false))
	v.Add(newNode("child4", "child3", false))
	if err := v.checkCycles(); err == nil {
		t.Errorf("Should have detected cycle")
	}
	if err := v.Validate(); err == nil {
		t.Errorf("Should have detected cycle")
	}
}
