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
			Input:        vt.ClusterConfigs(),
			ExpectOutput: vt.ClusterConfigs(),
		},
		{
			Name:  "inherit configs",
			Input: vt.AcmeRoot(),
			ExpectOutput: &ast.Root{
				ClusterObjects:         vt.AcmeCluster(),
				SystemObjects:          vt.System(),
				ClusterRegistryObjects: vt.ClusterRegistry(),
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(
						vt.AdminRoleBinding(),
						vt.AcmeResourceQuota(),
					),
					Children: []*ast.TreeNode{
						{
							Type: node.Namespace,
							Path: cmpath.FromSlash("namespaces/frontend"),
							Objects: vt.ObjectSets(
								vt.PodReaderRoleBinding(),
								vt.PodReaderRole(),
								vt.FrontendResourceQuota(),
								withName(vt.AdminRoleBinding(), "admin"),
								vt.AcmeResourceQuota(),
							),
						},
						{
							Type: node.Namespace,
							Path: cmpath.FromSlash("namespaces/frontend-test"),
							Objects: vt.ObjectSets(
								vt.DeploymentReaderRoleBinding(),
								vt.DeploymentReaderRole(),
								withName(vt.AdminRoleBinding(), "admin"),
								vt.AcmeResourceQuota(),
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
