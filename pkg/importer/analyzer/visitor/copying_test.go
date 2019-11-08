package visitor_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement"

	"github.com/pkg/errors"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
)

var copyingVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return visitor.NewCopying()
	},
	Options: func() []cmp.Option {
		return []cmp.Option{cmp.AllowUnexported(ast.FileObject{})}
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:         "empty",
			Input:        vt.Helper.EmptyRoot(),
			ExpectOutput: vt.Helper.EmptyRoot(),
		},
		{
			Name:         "cluster configs",
			Input:        vt.Helper.ClusterConfigs(),
			ExpectOutput: vt.Helper.ClusterConfigs(),
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
	visits []string
	errors []error
}

// VisitRoot implements Visitor
func (v *testVisitor) VisitRoot(c *ast.Root) *ast.Root {
	v.visits = append(v.visits, "Root")
	got := v.Visitor.VisitRoot(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitSystemObject implements Visitor
func (v *testVisitor) VisitSystemObject(c *ast.SystemObject) *ast.SystemObject {
	v.visits = append(v.visits, fmt.Sprintf("SystemObject %s %s", c.GroupVersionKind(), c.GetName()))
	got := v.Visitor.VisitSystemObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitClusterRegistryObject implements Visitor
func (v *testVisitor) VisitClusterRegistryObject(c *ast.ClusterRegistryObject) *ast.ClusterRegistryObject {
	v.visits = append(v.visits, fmt.Sprintf("ClusterRegistryObject %s %s", c.GroupVersionKind(), c.GetName()))
	got := v.Visitor.VisitClusterRegistryObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitClusterObject implements Visitor
func (v *testVisitor) VisitClusterObject(c *ast.ClusterObject) *ast.ClusterObject {
	v.visits = append(v.visits, fmt.Sprintf("ClusterObject %s %s", c.GroupVersionKind(), c.GetName()))
	got := v.Visitor.VisitClusterObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitTreeNode implements Visitor
func (v *testVisitor) VisitTreeNode(c *ast.TreeNode) *ast.TreeNode {
	v.visits = append(v.visits, fmt.Sprintf("TreeNode %s", c.Name()))
	got := v.Visitor.VisitTreeNode(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

// VisitObject implements Visitor
func (v *testVisitor) VisitObject(c *ast.NamespaceObject) *ast.NamespaceObject {
	v.visits = append(v.visits, fmt.Sprintf("NamespaceObject %s %s", c.GroupVersionKind(), c.GetName()))
	got := v.Visitor.VisitObject(c)
	if (c == got) == v.wantEq {
		v.errors = append(v.errors, errors.Errorf(v.fmt, "VisitRoot"))
	}
	return got
}

func (v *testVisitor) Check(t *testing.T) {
	t.Helper()

	expectOrder := []string{
		"Root",
		fmt.Sprintf("SystemObject %s/v1, Kind=Repo repo", configmanagement.GroupName),
		"ClusterRegistryObject /, Kind= ",
		fmt.Sprintf("ClusterObject rbac.authorization.k8s.io/v1, Kind=ClusterRole %s", vt.ClusterAdmin),
		fmt.Sprintf("ClusterObject rbac.authorization.k8s.io/v1, Kind=ClusterRoleBinding %s", vt.ClusterAdminBinding),
		"ClusterObject policy/v1beta1, Kind=PodSecurityPolicy example",
		"TreeNode namespaces",
		"NamespaceObject rbac.authorization.k8s.io/v1, Kind=RoleBinding admin",
		"NamespaceObject /v1, Kind=ResourceQuota quota",
		"TreeNode frontend",
		"NamespaceObject rbac.authorization.k8s.io/v1, Kind=RoleBinding admin",
		"NamespaceObject rbac.authorization.k8s.io/v1, Kind=Role pod-reader",
		"NamespaceObject /v1, Kind=ResourceQuota quota",
		"TreeNode frontend-test",
		"NamespaceObject rbac.authorization.k8s.io/v1, Kind=RoleBinding admin",
		"NamespaceObject rbac.authorization.k8s.io/v1, Kind=Role deployment-reader",
	}

	if diff := cmp.Diff(expectOrder, v.visits); diff != "" {
		t.Errorf("%#v", v.visits)
		t.Errorf("Invalid visit order:\n%s", diff)
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
