package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

func withName(o core.Object, name string) core.Object {
	o.SetName(name)
	return o
}

var inheritanceVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewInheritanceVisitor()
	},
	Options: func() []cmp.Option {
		return []cmp.Option{cmp.AllowUnexported(ast.FileObject{})}
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:         "preserve cluster configs",
			Input:        vt.Helper.ClusterConfigs(),
			ExpectOutput: vt.Helper.ClusterConfigs(),
		},
		{
			Name:  "inherit configs",
			Input: vt.Helper.AcmeRoot(),
			ExpectOutput: &ast.Root{
				ClusterObjects:         vt.Helper.AcmeCluster(),
				SystemObjects:          vt.Helper.System(),
				ClusterRegistryObjects: vt.Helper.ClusterRegistry(),
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(
						vt.Helper.AdminRoleBinding(),
						vt.Helper.AcmeResourceQuota(),
					),
					Children: []*ast.TreeNode{
						{
							Type: node.Namespace,
							Path: cmpath.FromSlash("namespaces/frontend"),
							Objects: vt.ObjectSets(
								vt.Helper.PodReaderRoleBinding(),
								vt.Helper.PodReaderRole(),
								vt.Helper.FrontendResourceQuota(),
								withName(vt.Helper.AdminRoleBinding(), "admin"),
								vt.Helper.AcmeResourceQuota(),
							),
						},
						{
							Type: node.Namespace,
							Path: cmpath.FromSlash("namespaces/frontend-test"),
							Objects: vt.ObjectSets(
								vt.Helper.DeploymentReaderRoleBinding(),
								vt.Helper.DeploymentReaderRole(),
								withName(vt.Helper.AdminRoleBinding(), "admin"),
								vt.Helper.AcmeResourceQuota(),
							),
						},
					},
				},
			},
		},
	},
}

func TestInheritanceVisitor(t *testing.T) {
	inheritanceVisitorTestcases.Run(t)
}
