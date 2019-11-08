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
	sel "github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors/seltest"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func withNamespaceSelector(o core.Object, selector string) core.Object {
	core.SetAnnotation(o, v1.NamespaceSelectorAnnotationKey, selector)
	return o
}

func withClusterSelector(o core.Object, selector string) core.Object {
	core.SetAnnotation(o, v1.ClusterSelectorAnnotationKey, selector)
	return o
}

func withClusterName(o core.Object, name string) core.Object {
	core.SetAnnotation(o, v1.ClusterNameAnnotationKey, name)
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
	seltest.Cluster("cluster-1", map[string]string{
		"env": "prod",
	}),
}
var selectors = []v1.ClusterSelector{
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
				cmp.AllowUnexported(ast.FileObject{}),
			}
		},
		Testcases: []vt.MutatingVisitorTestcase{
			{
				Name: "inline cluster selector annotation",
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
					Tree: &ast.TreeNode{
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
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
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
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
						Type: node.AbstractNamespace,
						Path: cmpath.FromSlash("namespaces"),
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
