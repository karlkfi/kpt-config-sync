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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors/seltest"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func withNamespaceSelector(o runtime.Object, selector string) runtime.Object {
	return annotate(o.(metav1.Object), v1.NamespaceSelectorAnnotationKey, selector).(runtime.Object)
}

func withClusterSelector(o runtime.Object, selector string) runtime.Object {
	return annotate(o.(metav1.Object), v1.ClusterSelectorAnnotationKey, selector).(runtime.Object)
}

func withClusterName(o runtime.Object, name string) runtime.Object {
	return annotate(o.(metav1.Object), v1.ClusterNameAnnotationKey, name).(runtime.Object)
}

func toJSON(s interface{}) string {
	j, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(j)
}

var clusters = []clusterregistry.Cluster{
	seltest.Cluster("cluster-1", map[string]string{
		"env": "prod",
	}),
}
var selectors = []v1alpha1.ClusterSelector{
	// Matches the cluster.
	seltest.Selector("sel-1",
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				"env": "prod",
			},
		}),
	// Does not match the cluster.
	seltest.Selector("sel-2",
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				"env": "test",
			},
		}),
}

func annotationInlinerVisitorTestcases(t *testing.T) vt.MutatingVisitorTestcases {
	return vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.Visitor {
			return NewAnnotationInlinerVisitor()
		},
		InitRoot: func(r *ast.Root) {
			cs, err := sel.NewClusterSelectors(clusters, selectors, "")
			if err != nil {
				t.Fatal(err)
			}
			sel.SetClusterSelector(cs, r)
		},
		Options: func() []cmp.Option {
			return []cmp.Option{
				cmpopts.IgnoreFields(ast.Root{}, "Data"),
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
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
			},
			{
				Name: "inline multiple objects",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
							withNamespaceSelector(vt.Helper.PodReaderRole(), "prod"),
							withNamespaceSelector(vt.Helper.AcmeResourceQuota(), "sensitive"),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{
							"prod":      &seltest.ProdNamespaceSelector,
							"sensitive": &seltest.SensitiveNamespaceSelector,
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
							withNamespaceSelector(vt.Helper.PodReaderRole(), toJSON(seltest.ProdNamespaceSelector)),
							withNamespaceSelector(vt.Helper.AcmeResourceQuota(), toJSON(seltest.SensitiveNamespaceSelector)),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{
							"prod":      &seltest.ProdNamespaceSelector,
							"sensitive": &seltest.SensitiveNamespaceSelector,
						},
					},
				},
			},
			{
				Name: "multiple policyspaces",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
						Children: []*ast.TreeNode{
							{
								Type:     node.AbstractNamespace,
								Relative: nomospath.NewRelative("namespaces/frontend"),
								Objects: vt.ObjectSets(
									withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
								),
								Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
							},
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
						Children: []*ast.TreeNode{
							{
								Type:     node.AbstractNamespace,
								Relative: nomospath.NewRelative("namespaces/frontend"),
								Objects: vt.ObjectSets(
									withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
								),
								Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
							},
						},
					},
				},
			},
			{
				Name: "NamespaceSelector missing",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
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
						Relative:  nomospath.NewRelative("namespaces"),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
						Children: []*ast.TreeNode{
							{
								Type:     node.AbstractNamespace,
								Relative: nomospath.NewRelative("namespaces/frontend"),
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
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Children: []*ast.TreeNode{
							{
								Type:      node.AbstractNamespace,
								Relative:  nomospath.NewRelative("namespaces/frontend"),
								Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
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
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding(),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding(),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
			},
			{
				Name: "NamespaceSelector in namespace",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.Namespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withNamespaceSelector(vt.Helper.AdminRoleBinding(), "prod"),
						),
						Selectors: map[string]*v1alpha1.NamespaceSelector{"prod": &seltest.ProdNamespaceSelector},
					},
				},
				ExpectErr: true,
			},
		},
	}
}

func TestNamespaceSelectorAnnotationInlinerVisitor(t *testing.T) {
	tc := annotationInlinerVisitorTestcases(t)
	tc.Run(t)
}

func TestClusterSelectorAnnotationInlinerVisitor(t *testing.T) {
	tests := vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.Visitor {
			return NewAnnotationInlinerVisitor()
		},
		InitRoot: func(r *ast.Root) {
			cs, err := sel.NewClusterSelectors(clusters, selectors, "cluster-1")
			if err != nil {
				t.Fatal(err)
			}
			sel.SetClusterSelector(cs, r)
		},
		Options: func() []cmp.Option {
			return []cmp.Option{
				cmpopts.IgnoreFields(ast.Root{}, "Data"),
			}
		},
		Testcases: []vt.MutatingVisitorTestcase{
			{
				Name: "inline cluster selector annotation",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: toJSON(selectors[0]),
							v1.ClusterNameAnnotationKey:     "cluster-1",
						},
					},
				},
			},
			{
				Name: "inline single object",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withClusterName(
								withClusterSelector(
									vt.Helper.AdminRoleBinding(),
									toJSON(selectors[0])),
								"cluster-1"),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: toJSON(selectors[0]),
							v1.ClusterNameAnnotationKey:     "cluster-1",
						},
					},
				},
			},
			{
				Name: "inline multiple objects",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
							withClusterSelector(vt.Helper.PodReaderRole(), "sel-1"),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type:     node.AbstractNamespace,
						Relative: nomospath.NewRelative("namespaces"),
						Objects: vt.ObjectSets(
							withClusterName(withClusterSelector(
								vt.Helper.AdminRoleBinding(), toJSON(selectors[0])), "cluster-1"),
							withClusterName(withClusterSelector(
								vt.Helper.PodReaderRole(), toJSON(selectors[0])), "cluster-1"),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: toJSON(selectors[0]),
							v1.ClusterNameAnnotationKey:     "cluster-1",
						},
					},
				},
			},
		},
	}
	tests.Run(t)
}
