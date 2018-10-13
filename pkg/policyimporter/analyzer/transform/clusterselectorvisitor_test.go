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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func TestClusterSelectorVisitor(t *testing.T) {
	clusters := []clusterregistry.Cluster{
		cluster("cluster-1", map[string]string{
			"env": "prod",
		}),
	}
	selectors := []v1alpha1.ClusterSelector{
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
	tests := vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.CheckingVisitor {
			return NewClusterSelectorVisitor()
		},
		InitRoot: func(r *ast.Root) {
			cs, err := NewClusterSelectors(clusters, selectors, "cluster-1")
			if err != nil {
				panic(err)
			}
			SetClusterSelector(cs, r)
		},
		Testcases: []vt.MutatingVisitorTestcase{
			{
				Name: "retain annotated namespace",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "retain namespace without annotation",
				Input: &ast.Root{
					Cluster: &ast.Cluster{
						Objects: vt.ClusterObjectSets(vt.Helper.NomosAdminClusterRole())},
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
					},
				},
				ExpectOutput: &ast.Root{
					Cluster: &ast.Cluster{
						Objects: vt.ClusterObjectSets(vt.Helper.NomosAdminClusterRole())},
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
					},
				},
			},
			{
				Name: "retain objects",
				Input: &ast.Root{
					Cluster: &ast.Cluster{
						Objects: vt.ClusterObjectSets(
							withClusterSelector(vt.Helper.NomosAdminClusterRole(), "sel-1"),
						)},
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
						),
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Cluster: &ast.Cluster{Objects: vt.ClusterObjectSets(
						withClusterSelector(vt.Helper.NomosAdminClusterRole(), "sel-1"),
					)},
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(
								vt.Helper.AdminRoleBinding(),
								"sel-1")),
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "retain objects without annotation",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding(),
						),
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding()),
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "filter out mis-targeted object",
				Input: &ast.Root{
					Cluster: &ast.Cluster{Objects: vt.ClusterObjectSets(
						withClusterSelector(vt.Helper.NomosAdminClusterRole(), "sel-2"),
					)},
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-2"),
						),
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Cluster: &ast.Cluster{}, // "sel-2" annotation not applied.
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "filter out namespaces intended for a different cluster",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: ast.AbstractNamespace,
						Path: "namespaces",
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
						),
						Annotations: map[string]string{
							v1alpha1.ClusterSelectorAnnotationKey: "sel-2",
						},
					},
				},
				ExpectOutput: &ast.Root{},
			},
		},
	}
	tests.Run(t)
}
