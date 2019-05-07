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
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/resourcequota"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func modQuota(q *corev1.ResourceQuota, name string, labels map[string]string, limits corev1.ResourceList) *corev1.ResourceQuota {
	nq := q.DeepCopy()
	nq.Name = name
	nq.Labels = labels
	nq.Spec.Hard = limits
	return nq
}

func modCluster(h *v1.HierarchicalQuota, c []*ast.ClusterObject) []*ast.ClusterObject {
	c = append(c, &ast.ClusterObject{
		FileObject: ast.FileObject{Object: h},
	})
	return c
}

func makeHierarchicalQuota(h *v1.HierarchicalQuotaNode) *v1.HierarchicalQuota {
	return &v1.HierarchicalQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "HierarchicalQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resourcequota.ResourceQuotaHierarchyName,
		},
		Spec: v1.HierarchicalQuotaSpec{
			Hierarchy: *h,
		},
	}
}

var quotaVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewQuotaVisitor()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:  "preserve cluster configs",
			Input: vt.Helper.ClusterConfigs(),
			ExpectOutput: &ast.Root{
				ClusterObjects: modCluster(makeHierarchicalQuota(&v1.HierarchicalQuotaNode{}), vt.Helper.AcmeCluster()),
				ImportToken:    vt.Helper.ImportToken,
				LoadTime:       vt.Helper.ImportTime,
			},
		},
		{
			Name:  "acme",
			Input: vt.Helper.AcmeRoot(),
			ExpectOutput: &ast.Root{
				ClusterObjects: modCluster(makeHierarchicalQuota(&v1.HierarchicalQuotaNode{
					Name: "namespaces",
					Type: "abstractNamespace",
					ResourceQuotaV1: modQuota(
						vt.Helper.AcmeResourceQuota(),
						resourcequota.ResourceQuotaObjectName,
						nil,
						corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5"),
						}),
					Children: []v1.HierarchicalQuotaNode{
						{
							ResourceQuotaV1: modQuota(
								vt.Helper.AcmeResourceQuota(),
								resourcequota.ResourceQuotaObjectName,
								resourcequota.NewConfigManagementQuotaLabels(),
								corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("5"),
									corev1.ResourceMemory: resource.MustParse("5"),
								}),
							Name: "frontend",
							Type: "namespace",
						},
						{
							ResourceQuotaV1: modQuota(
								vt.Helper.AcmeResourceQuota(),
								resourcequota.ResourceQuotaObjectName,
								resourcequota.NewConfigManagementQuotaLabels(),
								corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("5"),
								}),
							Name: "frontend-test",
							Type: "namespace",
						},
					}}),
					vt.Helper.AcmeCluster()),
				SystemObjects:          vt.Helper.System(),
				ClusterRegistryObjects: vt.Helper.ClusterRegistry(),
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(
						vt.Helper.AdminRoleBinding(),
						modQuota(
							vt.Helper.AcmeResourceQuota(),
							resourcequota.ResourceQuotaObjectName,
							nil,
							corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("5"),
							}),
					),
					Children: []*ast.TreeNode{
						{
							Type:        node.Namespace,
							Path:        cmpath.FromSlash("namespaces/frontend"),
							Labels:      map[string]string{"environment": "prod"},
							Annotations: map[string]string{"has-waffles": "true"},
							Objects: vt.ObjectSets(
								vt.Helper.PodReaderRoleBinding(),
								vt.Helper.PodReaderRole(),
								modQuota(
									vt.Helper.AcmeResourceQuota(),
									resourcequota.ResourceQuotaObjectName,
									resourcequota.NewConfigManagementQuotaLabels(),
									corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("5"),
										corev1.ResourceMemory: resource.MustParse("5"),
									}),
							),
						},
						{
							Type:        node.Namespace,
							Path:        cmpath.FromSlash("namespaces/frontend-test"),
							Labels:      map[string]string{"environment": "test"},
							Annotations: map[string]string{"has-waffles": "false"},
							Objects: vt.ObjectSets(
								vt.Helper.DeploymentReaderRoleBinding(),
								vt.Helper.DeploymentReaderRole(),
								modQuota(
									vt.Helper.AcmeResourceQuota(),
									resourcequota.ResourceQuotaObjectName,
									resourcequota.NewConfigManagementQuotaLabels(),
									corev1.ResourceList{
										corev1.ResourceCPU: resource.MustParse("5"),
									}),
							),
						},
					},
				},
			},
		},
		{
			Name: "skip abstract namespace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:    node.AbstractNamespace,
					Path:    cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(vt.Helper.AcmeResourceQuota()),
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Path: cmpath.FromSlash("namespaces/eng"),
							Children: []*ast.TreeNode{
								{
									Type: node.Namespace,
									Path: cmpath.FromSlash("namespaces/eng/frontend"),
									Objects: vt.ObjectSets(
										vt.Helper.FrontendResourceQuota(),
									),
								},
							},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				ClusterObjects: modCluster(makeHierarchicalQuota(&v1.HierarchicalQuotaNode{
					Name: "namespaces",
					Type: "abstractNamespace",
					ResourceQuotaV1: modQuota(
						vt.Helper.AcmeResourceQuota(),
						resourcequota.ResourceQuotaObjectName,
						nil,
						corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5"),
						}),
					Children: []v1.HierarchicalQuotaNode{
						{
							Name: "namespaces/eng",
							Type: "abstractNamespace",
							Children: []v1.HierarchicalQuotaNode{
								{
									ResourceQuotaV1: modQuota(
										vt.Helper.AcmeResourceQuota(),
										resourcequota.ResourceQuotaObjectName,
										resourcequota.NewConfigManagementQuotaLabels(),
										corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("5"),
											corev1.ResourceMemory: resource.MustParse("5"),
										}),
									Name: "frontend",
									Type: "namespace",
								},
							},
						}}}),
					nil),
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(
						modQuota(
							vt.Helper.AcmeResourceQuota(),
							resourcequota.ResourceQuotaObjectName,
							nil,
							corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("5"),
							}),
					),
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Path: cmpath.FromSlash("namespaces/eng"),
							Children: []*ast.TreeNode{
								{
									Type: node.Namespace,
									Path: cmpath.FromSlash("namespaces/eng/frontend"),
									Objects: vt.ObjectSets(
										modQuota(
											vt.Helper.AcmeResourceQuota(),
											resourcequota.ResourceQuotaObjectName,
											resourcequota.NewConfigManagementQuotaLabels(),
											corev1.ResourceList{
												corev1.ResourceCPU:    resource.MustParse("5"),
												corev1.ResourceMemory: resource.MustParse("5"),
											}),
									),
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "merge multiple quotas ",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:    node.AbstractNamespace,
					Path:    cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(vt.Helper.AcmeResourceQuota()),
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Path: cmpath.FromSlash("namespaces/eng"),
							Children: []*ast.TreeNode{
								{
									Type: node.Namespace,
									Path: cmpath.FromSlash("namespaces/eng/frontend"),
									Objects: vt.ObjectSets(
										modQuota(
											vt.Helper.FrontendResourceQuota(),
											"quota1",
											resourcequota.NewConfigManagementQuotaLabels(),
											corev1.ResourceList{
												corev1.ResourceCPU:     resource.MustParse("4"),
												corev1.ResourceStorage: resource.MustParse("6"),
											}),
										modQuota(
											vt.Helper.FrontendResourceQuota(),
											"quota2",
											resourcequota.NewConfigManagementQuotaLabels(),
											corev1.ResourceList{
												corev1.ResourceMemory:  resource.MustParse("2"),
												corev1.ResourceStorage: resource.MustParse("7"),
											}),
										modQuota(vt.Helper.FrontendResourceQuota(),
											"quota3",
											resourcequota.NewConfigManagementQuotaLabels(),
											corev1.ResourceList{
												corev1.ResourceCPU: resource.MustParse("3"),
											}),
									),
								},
							},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				ClusterObjects: modCluster(makeHierarchicalQuota(&v1.HierarchicalQuotaNode{
					Name: "namespaces",
					Type: "abstractNamespace",
					ResourceQuotaV1: modQuota(
						vt.Helper.AcmeResourceQuota(),
						resourcequota.ResourceQuotaObjectName,
						nil,
						corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5"),
						}),
					Children: []v1.HierarchicalQuotaNode{
						{
							Name: "namespaces/eng",
							Type: "abstractNamespace",
							Children: []v1.HierarchicalQuotaNode{
								{
									ResourceQuotaV1: modQuota(
										vt.Helper.AcmeResourceQuota(),
										resourcequota.ResourceQuotaObjectName,
										resourcequota.NewConfigManagementQuotaLabels(),
										corev1.ResourceList{
											corev1.ResourceCPU:     resource.MustParse("3"),
											corev1.ResourceMemory:  resource.MustParse("2"),
											corev1.ResourceStorage: resource.MustParse("6"),
										}),
									Name: "frontend",
									Type: "namespace",
								},
							},
						}}}),
					nil),
				Tree: &ast.TreeNode{
					Type: node.AbstractNamespace,
					Path: cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(
						modQuota(
							vt.Helper.AcmeResourceQuota(),
							resourcequota.ResourceQuotaObjectName,
							nil,
							corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("5"),
							}),
					),
					Children: []*ast.TreeNode{
						{
							Type: node.AbstractNamespace,
							Path: cmpath.FromSlash("namespaces/eng"),
							Children: []*ast.TreeNode{
								{
									Type: node.Namespace,
									Path: cmpath.FromSlash("namespaces/eng/frontend"),
									Objects: vt.ObjectSets(
										modQuota(
											vt.Helper.AcmeResourceQuota(),
											resourcequota.ResourceQuotaObjectName,
											resourcequota.NewConfigManagementQuotaLabels(),
											corev1.ResourceList{
												corev1.ResourceCPU:     resource.MustParse("3"),
												corev1.ResourceMemory:  resource.MustParse("2"),
												corev1.ResourceStorage: resource.MustParse("6"),
											}),
									),
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "no quota",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:    node.AbstractNamespace,
					Path:    cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(vt.Helper.AdminRoleBinding()),
					Children: []*ast.TreeNode{
						{
							Type:    node.Namespace,
							Path:    cmpath.FromSlash("namespaces/frontend"),
							Objects: vt.ObjectSets(vt.Helper.PodReaderRoleBinding(), vt.Helper.PodReaderRole()),
						},
						{
							Type:    node.Namespace,
							Path:    cmpath.FromSlash("namespaces/frontend-test"),
							Objects: vt.ObjectSets(vt.Helper.DeploymentReaderRoleBinding(), vt.Helper.DeploymentReaderRole()),
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				ClusterObjects: modCluster(makeHierarchicalQuota(&v1.HierarchicalQuotaNode{
					Name: "namespaces",
					Type: "abstractNamespace",
					Children: []v1.HierarchicalQuotaNode{
						{
							Name: "frontend",
							Type: "namespace",
						}, {
							Name: "frontend-test",
							Type: "namespace",
						}},
				}), nil),
				Tree: &ast.TreeNode{
					Type:    node.AbstractNamespace,
					Path:    cmpath.FromSlash("namespaces"),
					Objects: vt.ObjectSets(vt.Helper.AdminRoleBinding()),
					Children: []*ast.TreeNode{
						{
							Type:    node.Namespace,
							Path:    cmpath.FromSlash("namespaces/frontend"),
							Objects: vt.ObjectSets(vt.Helper.PodReaderRoleBinding(), vt.Helper.PodReaderRole()),
						},
						{
							Type:    node.Namespace,
							Path:    cmpath.FromSlash("namespaces/frontend-test"),
							Objects: vt.ObjectSets(vt.Helper.DeploymentReaderRoleBinding(), vt.Helper.DeploymentReaderRole()),
						},
					},
				},
			},
		},
	},
}

func TestQuotaVisitor(t *testing.T) {
	quotaVisitorTestcases.Run(t)
}
