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
	"encoding/json"
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func withNamespaceSelector(o runtime.Object, selector string) runtime.Object {
	return annotate(o, v1alpha1.NamespaceSelectorAnnotationKey, selector)
}

func withClusterSelector(o runtime.Object, selector string) runtime.Object {
	return annotate(o, v1alpha1.ClusterSelectorAnnotationKey, selector)
}

func annotate(o runtime.Object, key, annotation string) runtime.Object {
	m := o.(metav1.Object)
	a := m.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = annotation
	m.SetAnnotations(a)
	return o
}

func toJSON(s interface{}) string {
	j, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(j)
}

var clusters = []clusterregistry.Cluster{
	cluster("cluster-1", map[string]string{
		"env": "prod",
	}),
}
var selectors = []v1alpha1.ClusterSelector{
	// Matches the cluster.
	selector("sel-1",
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				"env": "prod",
			},
		}),
	// Does not match the cluster.
	selector("sel-2",
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				"env": "test",
			},
		}),
}
var annotationInlinerVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.CheckingVisitor {
		cs, err := NewClusterSelectors(clusters, selectors)
		if err != nil {
			panic(err)
		}
		return NewAnnotationInlinerVisitor(cs)
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
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
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
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						withNamespaceSelector(vt.Helper.PodReaderRole(), "prod"),
						withNamespaceSelector(vt.Helper.AcmeResourceQuota(), "sensitive"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{
						"prod":      &prodNamespaceSelector,
						"sensitive": &sensitiveNamespaceSelector,
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
						withNamespaceSelector(vt.Helper.PodReaderRole(), toJSON(prodNamespaceSelector)),
						withNamespaceSelector(vt.Helper.AcmeResourceQuota(), toJSON(sensitiveNamespaceSelector)),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{
						"prod":      &prodNamespaceSelector,
						"sensitive": &sensitiveNamespaceSelector,
					},
				},
			},
		},
		{
			Name: "inline multiple objects (cluster selector)",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
						withClusterSelector(vt.Helper.PodReaderRole(), "sel-1"),
						withClusterSelector(vt.Helper.AcmeResourceQuota(), "sel-2"),
					),
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withClusterSelector(
							vt.Helper.AdminRoleBinding(), toJSON(selectors[0])),
						withClusterSelector(
							vt.Helper.PodReaderRole(), toJSON(selectors[0])),
						withClusterSelector(vt.Helper.AcmeResourceQuota(), toJSON(selectors[1])),
					),
				},
			},
		},
		{
			Name: "multiple policyspaces",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type: ast.Policyspace,
							Path: "namespaces/frontend",
							Objects: vt.ObjectSets(
								withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
							),
							Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type: ast.Policyspace,
							Path: "namespaces/frontend",
							Objects: vt.ObjectSets(
								withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(prodNamespaceSelector)),
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
					Path: "namespaces",
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
					Type:      ast.Policyspace,
					Path:      "namespaces",
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type: ast.Policyspace,
							Path: "namespaces/frontend",
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
					Type: ast.Policyspace,
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:      ast.Policyspace,
							Path:      "namespaces/frontend",
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
					Path: "namespaces",
					Objects: vt.ObjectSets(
						vt.Helper.AdminRoleBinding(),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "namespaces",
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
					Path: "namespaces",
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
					),
					Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &prodNamespaceSelector},
				},
			},
			ExpectErr: true,
		},
	},
}

func TestNamespaceSelectorAnnotationInlinerVisitor(t *testing.T) {
	annotationInlinerVisitorTestcases.Run(t)
}

func TestClusterSelectorAnnotationInlinerVisitor(t *testing.T) {
	tests := vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.CheckingVisitor {
			cs, err := NewClusterSelectors(clusters, selectors)
			if err != nil {
				panic(err)
			}
			return NewAnnotationInlinerVisitor(cs)
		},
		Testcases: []vt.MutatingVisitorTestcase{
			{
				Name: "inline namespace annotations",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.Policyspace,
						Path: "namespaces",
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.Policyspace,
						Path: "namespaces",
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: toJSON(selectors[0]),
						},
					},
				},
			},
			{
				Name: "inline single object",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.Policyspace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
						),
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.Policyspace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), toJSON(selectors[0])),
						),
					},
				},
			},
			{
				Name: "inline multiple objects",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.Policyspace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
							withClusterSelector(vt.Helper.PodReaderRole(), "sel-1"),
							withClusterSelector(vt.Helper.AcmeResourceQuota(), "sel-2"),
						),
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.Policyspace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(
								vt.Helper.AdminRoleBinding(), toJSON(selectors[0])),
							withClusterSelector(
								vt.Helper.PodReaderRole(), toJSON(selectors[0])),
							withClusterSelector(vt.Helper.AcmeResourceQuota(), toJSON(selectors[1])),
						),
					},
				},
			},
		},
	}
	tests.Run(t)
}
