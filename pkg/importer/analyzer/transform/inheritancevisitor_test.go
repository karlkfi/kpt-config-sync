package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors/seltest"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func withName(o core.Object, name string) core.Object {
	o.SetName(name)
	return o
}

var inheritanceVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewInheritanceVisitor(
			map[schema.GroupKind]*InheritanceSpec{
				kinds.RoleBinding().GroupKind(): {
					Mode: "inherit",
				},
				kinds.ResourceQuota().GroupKind(): {
					Mode: "inherit",
				},
			},
		)
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
					Children: []*ast.TreeNode{
						{
							Type:        node.Namespace,
							Path:        cmpath.FromSlash("namespaces/frontend"),
							Labels:      map[string]string{"environment": "prod"},
							Annotations: map[string]string{"has-waffles": "true"},
							Objects: vt.ObjectSets(
								vt.Helper.PodReaderRoleBinding(),
								vt.Helper.PodReaderRole(),
								vt.Helper.FrontendResourceQuota(),
								withName(vt.Helper.AdminRoleBinding(), "admin"),
								vt.Helper.AcmeResourceQuota(),
							),
						},
						{
							Type:        node.Namespace,
							Path:        cmpath.FromSlash("namespaces/frontend-test"),
							Labels:      map[string]string{"environment": "test"},
							Annotations: map[string]string{"has-waffles": "false"},
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
		{
			Name: "inherit filtered by NamespaceSelector",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
					),
					Children: []*ast.TreeNode{
						{
							Type:   node.Namespace,
							Path:   cmpath.FromSlash("namespaces/frontend"),
							Labels: map[string]string{"env": "prod"},
						},
						{
							Type:   node.Namespace,
							Path:   cmpath.FromSlash("namespaces/frontend-test"),
							Labels: map[string]string{"env": "test"},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:   node.Namespace,
							Path:   cmpath.FromSlash("namespaces/frontend"),
							Labels: map[string]string{"env": "prod"},
							Objects: vt.ObjectSets(
								withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
							),
						},
						{
							Type:   node.Namespace,
							Path:   cmpath.FromSlash("namespaces/frontend-test"),
							Labels: map[string]string{"env": "test"},
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
