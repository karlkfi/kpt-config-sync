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

package visitor_test

import (
	"testing"

	"github.com/pkg/errors"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

var copyingVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return visitor.NewCopying()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:         "empty",
			Input:        vt.Helper.EmptyRoot(),
			ExpectOutput: vt.Helper.EmptyRoot(),
		},
		{
			Name:         "cluster policies",
			Input:        vt.Helper.ClusterPolicies(),
			ExpectOutput: vt.Helper.ClusterPolicies(),
		},
		{
			Name:         "acme",
			Input:        vt.Helper.AcmeRoot(),
			ExpectOutput: vt.Helper.AcmeRoot(),
		},
	},
}

func TestCopyingVisitor(t *testing.T) {
	t.Run("copyingvisitor", copyingVisitorTestcases.Run)
}

type testVisitor struct {
	ast.Visitor

	wantEq bool
	fmt    string

	RootVisited                  bool
	SystemVisited                bool
	SystemObjectVisited          bool
	ClusterRegistryVisited       bool
	ClusterRegistryObjectVisited bool
	ClusterVisited               bool
	ClusterObjectVisited         bool
	TreeNodeVisited              bool
	ObjectVisited                bool

	errors []error
}

// VisitRoot implements Visitor
func (v *testVisitor) VisitRoot(c *ast.Root) *ast.Root {
	v.RootVisited = true
	got := v.Visitor.VisitRoot(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitSystem implements Visitor
func (v *testVisitor) VisitSystem(c *ast.System) *ast.System {
	v.SystemVisited = true
	got := v.Visitor.VisitSystem(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitSystemObject implements Visitor
func (v *testVisitor) VisitSystemObject(c *ast.SystemObject) *ast.SystemObject {
	v.SystemObjectVisited = true
	got := v.Visitor.VisitSystemObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitClusterRegistry implements Visitor
func (v *testVisitor) VisitClusterRegistry(c *ast.ClusterRegistry) *ast.ClusterRegistry {
	v.ClusterRegistryVisited = true
	got := v.Visitor.VisitClusterRegistry(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitClusterRegistryObject implements Visitor
func (v *testVisitor) VisitClusterRegistryObject(c *ast.ClusterRegistryObject) *ast.ClusterRegistryObject {
	v.ClusterRegistryObjectVisited = true
	got := v.Visitor.VisitClusterRegistryObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitCluster implements Visitor
func (v *testVisitor) VisitCluster(c *ast.Cluster) *ast.Cluster {
	v.ClusterVisited = true
	got := v.Visitor.VisitCluster(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitClusterObject implements Visitor
func (v *testVisitor) VisitClusterObject(c *ast.ClusterObject) *ast.ClusterObject {
	v.ClusterObjectVisited = true
	got := v.Visitor.VisitClusterObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitTreeNode implements Visitor
func (v *testVisitor) VisitTreeNode(c *ast.TreeNode) *ast.TreeNode {
	v.TreeNodeVisited = true
	got := v.Visitor.VisitTreeNode(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitObject implements Visitor
func (v *testVisitor) VisitObject(c *ast.NamespaceObject) *ast.NamespaceObject {
	v.ObjectVisited = true
	got := v.Visitor.VisitObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

func (v *testVisitor) Check(t *testing.T) {
	t.Helper()

	if !v.RootVisited {
		t.Errorf("Root not visited")
	}
	if !v.SystemVisited {
		t.Errorf("System not visited")
	}
	if !v.SystemObjectVisited {
		t.Errorf("SystemObject not visited")
	}
	if !v.ClusterRegistryVisited {
		t.Errorf("ClusterRegistry not visited")
	}
	if !v.ClusterRegistryObjectVisited {
		t.Errorf("ClusterRegistryObject not visited")
	}
	if !v.ClusterVisited {
		t.Errorf("Cluster not visited")
	}
	if !v.ClusterObjectVisited {
		t.Errorf("ClusterObject not visited")
	}
	if !v.TreeNodeVisited {
		t.Errorf("TreeNode not visited")
	}
	if !v.ObjectVisited {
		t.Errorf("Object not visited")
	}
	for _, err := range v.errors {
		t.Errorf("Got error: %s", err)
	}
}

func TestCopyingVisitorCopies(t *testing.T) {
	v := &testVisitor{
		fmt:    "%s did not copy (pointers equal)",
		wantEq: true,
	}
	c := visitor.NewCopying()
	c.SetImpl(v)
	v.Visitor = c

	input := vt.Helper.AcmeRoot()
	out := input.Accept(v)
	if out == input {
		t.Errorf("ouptut and input have same pointer value")
	}
	v.Check(t)
}
