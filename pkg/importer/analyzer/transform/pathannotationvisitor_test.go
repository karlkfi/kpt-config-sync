package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/testing/fake"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	rbacv1 "k8s.io/api/rbac/v1"
)

func withSourceAnnotation(o core.Object, annotation string) core.Object {
	core.SetAnnotation(o, v1.SourcePathAnnotationKey, annotation)
	return o
}

func adminRoleBindingWithAnnotation() *rbacv1.RoleBinding {
	rb := vt.AdminRoleBinding()
	rb.Annotations = map[string]string{"color": "blue"}
	return rb
}

var pathAnnotationVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewPathAnnotationVisitor()
	},
	Options: func() []cmp.Option {
		return []cmp.Option{
			cmp.AllowUnexported(ast.FileObject{}),
		}
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name: "annotate abstract namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
				},
			},
		},
		{
			Name: "annotate namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Children: []*ast.TreeNode{
						{
							Type:     node.Namespace,
							Relative: cmpath.RelativeSlash("namespaces/bar"),
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/bar")},
							},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Children: []*ast.TreeNode{
						{
							Type:     node.Namespace,
							Relative: cmpath.RelativeSlash("namespaces/bar"),
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/bar",
									core.Annotation(v1.SourcePathAnnotationKey, "namespaces/bar/namespace.yaml"))},
							},
						},
					},
				},
			},
		},
		{
			Name: "annotate RoleBinding in abstract namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(vt.AdminRoleBinding(), cmpath.RelativeSlash("acme/admin.yaml")),
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(withSourceAnnotation(vt.AdminRoleBinding(), "acme/admin.yaml"), cmpath.RelativeSlash("acme/admin.yaml")),
					),
				},
			},
		},
		{
			Name: "annotate RoleBinding in namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Children: []*ast.TreeNode{
						{
							Type:     node.Namespace,
							Relative: cmpath.RelativeSlash("namespaces/bar"),
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/bar")},
								{FileObject: fake.RoleAtPath("namespaces/bar/rb.yaml")},
							},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Children: []*ast.TreeNode{
						{
							Type:     node.Namespace,
							Relative: cmpath.RelativeSlash("namespaces/bar"),
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/bar",
									core.Annotation(v1.SourcePathAnnotationKey, "namespaces/bar/namespace.yaml"))},
								{FileObject: fake.RoleAtPath("namespaces/bar/rb.yaml",
									core.Annotation(v1.SourcePathAnnotationKey, "namespaces/bar/rb.yaml"))},
							},
						},
					},
				},
			},
		},
		{
			Name: "preserve annotations",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.Namespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(adminRoleBindingWithAnnotation(), cmpath.RelativeSlash("acme/admin.yaml")),
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.Namespace,
					Relative: cmpath.RelativeSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(withSourceAnnotation(adminRoleBindingWithAnnotation(), "acme/admin.yaml"), cmpath.RelativeSlash("acme/admin.yaml")),
					),
				},
			},
		},
	},
}

func TestPathAnnotationVisitor(t *testing.T) {
	pathAnnotationVisitorTestcases.Run(t)
}
