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

	"encoding/json"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func withSelectorAnnotation(o runtime.Object, annotation string) runtime.Object {
	m := o.(metav1.Object)
	a := m.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[v1alpha1.NamespaceSelectorAnnotationKey] = annotation
	m.SetAnnotations(a)
	return o
}

func toJSON(s v1alpha1.NamespaceSelector) string {
	j, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(j)
}

var annotationInlinerVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.CheckingVisitor {
		return NewAnnotationInlinerVisitor()
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
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
		},
		{
			Name: "inline multiple objects",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
						withSelectorAnnotation(vt.Helper.PodReaderRole(), "prod"),
						withSelectorAnnotation(vt.Helper.AcmeResourceQuota(), "sensitive"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector, "sensitive": &sensitiveNamespaceSelector},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
						withSelectorAnnotation(vt.Helper.PodReaderRole(), toJSON(prodNamespaceSelector)),
						withSelectorAnnotation(vt.Helper.AcmeResourceQuota(), toJSON(sensitiveNamespaceSelector)),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector, "sensitive": &sensitiveNamespaceSelector},
				},
			},
		},
		{
			Name: "multiple policyspaces",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type: ast.Policyspace,
							Path: "acme/frontend",
							Objects: vt.ObjectSets(
								withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
							),
							Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type: ast.Policyspace,
							Path: "acme/frontend",
							Objects: vt.ObjectSets(
								withSelectorAnnotation(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
							),
							Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
						},
					},
				},
			},
		},
		{
			Name: "NamespaceSelector missing",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
					),
				},
			},
			ExpectErr: true,
		},
		{
			Name: "NamespaceSelector scoped to same directory 1",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:      ast.Policyspace,
					Path:      "acme",
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type: ast.Policyspace,
							Path: "acme/frontend",
							Objects: vt.ObjectSets(
								withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
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
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:      ast.Policyspace,
							Path:      "acme/frontend",
							Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
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
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						vt.Helper.AdminRoleBinding(),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Objects: vt.ObjectSets(
						vt.Helper.AdminRoleBinding(),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
		},
		{
			Name: "NamespaceSelector in namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Namespace,
					Path: "acme",
					Objects: vt.ObjectSets(
						withSelectorAnnotation(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
			ExpectErr: true,
		},
	},
}

func TestAnnotationInlinerVisitor(t *testing.T) {
	annotationInlinerVisitorTestcases.Run(t)
}
