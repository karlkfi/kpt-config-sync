/*
Copyright 2018 The CSP Config Management Authors.

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

package transform

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func withSourceAnnotation(o runtime.Object, annotation string) runtime.Object {
	m := o.(metav1.Object)
	a := m.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[v1.SourcePathAnnotationKey] = annotation
	m.SetAnnotations(a)
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
