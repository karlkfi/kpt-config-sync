package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/object"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func withSourceAnnotation(o runtime.Object, annotation string) runtime.Object {
	object.SetAnnotation(o.(metav1.Object), v1.SourcePathAnnotationKey, annotation)
	return o
}

func adminRoleBindingWithAnnotation() *rbacv1.RoleBinding {
	rb := vt.Helper.AdminRoleBinding()
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
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        node.AbstractNamespace,
					Path:        cmpath.FromSlash("namespaces"),
					Annotations: map[string]string{v1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "annotate namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: node.Namespace,
					Path: cmpath.FromSlash("namespaces"),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        node.Namespace,
					Path:        cmpath.FromSlash("namespaces"),
					Annotations: map[string]string{v1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "annotate RoleBinding in abstract namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(vt.Helper.AdminRoleBinding(), cmpath.FromSlash("acme/admin.yaml")),
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(withSourceAnnotation(vt.Helper.AdminRoleBinding(), "acme/admin.yaml"), cmpath.FromSlash("acme/admin.yaml")),
					),
					Annotations: map[string]string{v1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "annotate RoleBinding in namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: node.Namespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(vt.Helper.AdminRoleBinding(), cmpath.FromSlash("acme/admin.yaml")),
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: node.Namespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.FileObjectSets(
						ast.NewFileObject(withSourceAnnotation(vt.Helper.AdminRoleBinding(), "acme/admin.yaml"), cmpath.FromSlash("acme/admin.yaml")),
					),
					Annotations: map[string]string{v1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "preserve annotations",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        node.Namespace,
					Path:        cmpath.FromSlash("namespaces"),
					Annotations: map[string]string{"color": "orange"},
					Objects: vt.FileObjectSets(
						ast.NewFileObject(adminRoleBindingWithAnnotation(), cmpath.FromSlash("acme/admin.yaml")),
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        node.Namespace,
					Path:        cmpath.FromSlash("namespaces"),
					Annotations: map[string]string{"color": "orange", v1.SourcePathAnnotationKey: "namespaces"},
					Objects: vt.FileObjectSets(
						ast.NewFileObject(withSourceAnnotation(adminRoleBindingWithAnnotation(), "acme/admin.yaml"), cmpath.FromSlash("acme/admin.yaml")),
					),
				},
			},
		},
	},
}

func TestPathAnnotationVisitor(t *testing.T) {
	pathAnnotationVisitorTestcases.Run(t)
}
