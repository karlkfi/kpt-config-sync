package transform

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors/seltest"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

func withNamespaceSelector(o core.Object, selector string) core.Object {
	core.SetAnnotation(o, v1.NamespaceSelectorAnnotationKey, selector)
	return o
}

func toJSON(s interface{}) string {
	j, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(j)
}

func annotationInlinerVisitorTestcases() vt.MutatingVisitorTestcases {
	return vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.Visitor {
			return NewAnnotationInlinerVisitor()
		},
		InitRoot: func(r *ast.Root) {},
		Options: func() []cmp.Option {
			return []cmp.Option{
				cmpopts.IgnoreFields(ast.Root{}, "Data"),
				cmp.AllowUnexported(ast.FileObject{}),
			}
		},
		Testcases: []vt.MutatingVisitorTestcase{
			{
				Name:         "preserve acme",
				Input:        vt.Helper.AcmeRoot(),
				ExpectOutput: vt.Helper.AcmeRoot(),
			},
			{
				Name: "inline single object",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
						),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
			},
			{
				Name: "inline multiple objects",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
							withNamespaceSelector(vt.Helper.PodReaderRole(), "prod"),
							withNamespaceSelector(vt.Helper.AcmeResourceQuota(), "sensitive"),
						),
						Selectors: map[string]*v1.NamespaceSelector{
							"prod":      &seltest.ProdNamespaceSelector,
							"sensitive": &seltest.SensitiveNamespaceSelector,
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
							withNamespaceSelector(vt.Helper.PodReaderRole(), toJSON(seltest.ProdNamespaceSelector)),
							withNamespaceSelector(vt.Helper.AcmeResourceQuota(), toJSON(seltest.SensitiveNamespaceSelector)),
						),
						Selectors: map[string]*v1.NamespaceSelector{
							"prod":      &seltest.ProdNamespaceSelector,
							"sensitive": &seltest.SensitiveNamespaceSelector,
						},
					},
				},
			},
			{
				Name: "multiple abstract namespaces",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
						Children: []*ast.TreeNode{
							{
								Type: node.AbstractNamespace,
								Path: cmpath.FromSlash("namespaces/frontend"),
								Objects: vt.ObjectSets(
									withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
								),
								Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
							},
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
						),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
						Children: []*ast.TreeNode{
							{
								Type: node.AbstractNamespace,
								Path: cmpath.FromSlash("namespaces/frontend"),
								Objects: vt.ObjectSets(
									withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
								),
								Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
							},
						},
					},
				},
			},
			{
				Name: "NamespaceSelector missing",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
					},
				},
				ExpectErr: true,
			},
			{
				Name: "NamespaceSelector scoped to same directory 1",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:      node.AbstractNamespace,
						Path:      cmpath.FromSlash("namespaces"),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
						Children: []*ast.TreeNode{
							{
								Type: node.AbstractNamespace,
								Path: cmpath.FromSlash("namespaces/frontend"),
								Objects: vt.ObjectSets(
									withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
								),
							},
						},
					},
				},
				ExpectErr: true,
			},
			{
				Name: "NamespaceSelector scoped to same directory 2",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Children: []*ast.TreeNode{
							{
								Type:      node.AbstractNamespace,
								Path:      cmpath.FromSlash("namespaces/frontend"),
								Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
							},
						},
					},
				},
				ExpectErr: true,
			},
			{
				Name: "NamespaceSelector unused",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding(),
						),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding(),
						),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
			},
			{
				Name: "NamespaceSelector in namespace",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.Namespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Selectors: map[string]*v1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
				ExpectErr: true,
			},
		},
	}
}

func TestNamespaceSelectorAnnotationInlinerVisitor(t *testing.T) {
	tc := annotationInlinerVisitorTestcases()
	tc.Run(t)
}
