package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	sel "github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors/seltest"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func TestClusterSelectorVisitor(t *testing.T) {
	clusters := []clusterregistry.Cluster{
		seltest.Cluster("cluster-1", map[string]string{
			"env": "prod",
		}),
	}
	selectors := []v1.ClusterSelector{
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
	tests := vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.Visitor {
			return NewClusterSelectorVisitor()
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
				Name: "retain annotated namespace",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "retain namespace without annotation",
				Input: &ast.Root{
					ClusterObjects: vt.ClusterObjectSets(vt.Helper.NomosAdminClusterRole()),
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
					},
				},
				ExpectOutput: &ast.Root{
					ClusterObjects: vt.ClusterObjectSets(vt.Helper.NomosAdminClusterRole()),
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
					},
				},
			},
			{
				Name: "retain objects",
				Input: &ast.Root{
					ClusterObjects: vt.ClusterObjectSets(
						withClusterSelector(vt.Helper.NomosAdminClusterRole(), "sel-1"),
					),
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					ClusterObjects: vt.ClusterObjectSets(
						withClusterSelector(vt.Helper.NomosAdminClusterRole(), "sel-1"),
					),
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withClusterSelector(
								vt.Helper.AdminRoleBinding(),
								"sel-1")),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "retain objects without annotation",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding(),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							vt.Helper.AdminRoleBinding()),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "filter out mis-targeted object",
				Input: &ast.Root{
					ClusterObjects: vt.ClusterObjectSets(
						withClusterSelector(vt.Helper.NomosAdminClusterRole(), "sel-2"),
					),
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-2"),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
				ExpectOutput: &ast.Root{
					// "sel-2" annotation not applied.
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-1",
						},
					},
				},
			},
			{
				Name: "filter out namespaces intended for a different cluster",
				Input: &ast.Root{
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
						Objects: vt.ObjectSets(
							withClusterSelector(vt.Helper.AdminRoleBinding(), "sel-1"),
						),
						Annotations: map[string]string{
							v1.ClusterSelectorAnnotationKey: "sel-2",
						},
					},
				},
				ExpectOutput: &ast.Root{},
			},
		},
	}
	tests.Run(t)
}
