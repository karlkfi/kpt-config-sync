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

package transform

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func withSourceAnnotation(o runtime.Object, annotation string) runtime.Object {
	m := o.(metav1.Object)
	a := m.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[v1alpha1.SourcePathAnnotationKey] = annotation
	m.SetAnnotations(a)
	return o
}

func adminRoleBindingWithAnnotation() *v1.RoleBinding {
	rb := vt.Helper.AdminRoleBinding()
	rb.Annotations = map[string]string{"color": "blue"}
	return rb
}

var pathAnnotationVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.CheckingVisitor {
		return NewPathAnnotationVisitor()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name: "annotate policyspace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        ast.Policyspace,
					Path:        "namespaces",
					Annotations: map[string]string{v1alpha1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "annotate namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Namespace,
					Path: "namespaces",
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        ast.Namespace,
					Path:        "namespaces",
					Annotations: map[string]string{v1alpha1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "annotate RoleBinding in policyspace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.FileObjectSets(
						ast.FileObject{Object: vt.Helper.AdminRoleBinding(), Source: "acme/admin.yaml"},
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.FileObjectSets(
						ast.FileObject{Object: withSourceAnnotation(vt.Helper.AdminRoleBinding(), "acme/admin.yaml"), Source: "acme/admin.yaml"},
					),
					Annotations: map[string]string{v1alpha1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "annotate RoleBinding in namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Namespace,
					Path: "namespaces",
					Objects: vt.FileObjectSets(
						ast.FileObject{Object: vt.Helper.AdminRoleBinding(), Source: "acme/admin.yaml"},
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Namespace,
					Path: "namespaces",
					Objects: vt.FileObjectSets(
						ast.FileObject{Object: withSourceAnnotation(vt.Helper.AdminRoleBinding(), "acme/admin.yaml"), Source: "acme/admin.yaml"},
					),
					Annotations: map[string]string{v1alpha1.SourcePathAnnotationKey: "namespaces"},
				},
			},
		},
		{
			Name: "preserve annotations",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        ast.Namespace,
					Path:        "namespaces",
					Annotations: map[string]string{"color": "orange"},
					Objects: vt.FileObjectSets(
						ast.FileObject{Object: adminRoleBindingWithAnnotation(), Source: "acme/admin.yaml"},
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:        ast.Namespace,
					Path:        "namespaces",
					Annotations: map[string]string{"color": "orange", v1alpha1.SourcePathAnnotationKey: "namespaces"},
					Objects: vt.FileObjectSets(
						ast.FileObject{Object: withSourceAnnotation(adminRoleBindingWithAnnotation(), "acme/admin.yaml"), Source: "acme/admin.yaml"},
					),
				},
			},
		},
	},
}

func TestPathAnnotationVisitor(t *testing.T) {
	pathAnnotationVisitorTestcases.Run(t)
}
