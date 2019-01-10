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

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/resourcequota"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func modQuota(q *corev1.ResourceQuota, name string, labels map[string]string, limits corev1.ResourceList) *corev1.ResourceQuota {
	nq := q.DeepCopy()
	nq.Name = name
	nq.Labels = labels
	nq.Spec.Hard = limits
	return nq
}

var quotaVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewQuotaVisitor()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:         "preserve cluster policies",
			Input:        vt.Helper.ClusterPolicies(),
			ExpectOutput: vt.Helper.ClusterPolicies(),
		},
		{
			Name:  "acme",
			Input: vt.Helper.AcmeRoot(),
			ExpectOutput: &ast.Root{
				Cluster: vt.Helper.AcmeCluster(),
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: nomospath.NewFakeRelative("namespaces"),
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
							Relative:    nomospath.NewFakeRelative("namespaces/frontend"),
							Labels:      map[string]string{"environment": "prod"},
							Annotations: map[string]string{"has-waffles": "true"},
							Objects: vt.ObjectSets(
								vt.Helper.PodReaderRoleBinding(),
								vt.Helper.PodReaderRole(),
								modQuota(
									vt.Helper.AcmeResourceQuota(),
									resourcequota.ResourceQuotaObjectName,
									resourcequota.NewNomosQuotaLabels(),
									corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("5"),
										corev1.ResourceMemory: resource.MustParse("5"),
									}),
							),
						},
						{
							Type:        node.Namespace,
							Relative:    nomospath.NewFakeRelative("namespaces/frontend-test"),
							Labels:      map[string]string{"environment": "test"},
							Annotations: map[string]string{"has-waffles": "false"},
							Objects: vt.ObjectSets(
								vt.Helper.DeploymentReaderRoleBinding(),
								vt.Helper.DeploymentReaderRole(),
								modQuota(
									vt.Helper.AcmeResourceQuota(),
									resourcequota.ResourceQuotaObjectName,
									resourcequota.NewNomosQuotaLabels(),
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
			Name: "skip policyspace",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: nomospath.NewFakeRelative("namespaces"),
					Objects:  vt.ObjectSets(vt.Helper.AcmeResourceQuota()),
					Children: []*ast.TreeNode{
						{
							Type:     node.AbstractNamespace,
							Relative: nomospath.NewFakeRelative("namespaces/eng"),
							Children: []*ast.TreeNode{
								{
									Type:     node.Namespace,
									Relative: nomospath.NewFakeRelative("namespaces/eng/frontend"),
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
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: nomospath.NewFakeRelative("namespaces"),
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
							Type:     node.AbstractNamespace,
							Relative: nomospath.NewFakeRelative("namespaces/eng"),
							Children: []*ast.TreeNode{
								{
									Type:     node.Namespace,
									Relative: nomospath.NewFakeRelative("namespaces/eng/frontend"),
									Objects: vt.ObjectSets(
										modQuota(
											vt.Helper.AcmeResourceQuota(),
											resourcequota.ResourceQuotaObjectName,
											resourcequota.NewNomosQuotaLabels(),
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
			Name: "no quota",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: nomospath.NewFakeRelative("namespaces"),
					Objects:  vt.ObjectSets(vt.Helper.AdminRoleBinding()),
					Children: []*ast.TreeNode{
						{
							Type:     node.Namespace,
							Relative: nomospath.NewFakeRelative("namespaces/frontend"),
							Objects:  vt.ObjectSets(vt.Helper.PodReaderRoleBinding(), vt.Helper.PodReaderRole()),
						},
						{
							Type:     node.Namespace,
							Relative: nomospath.NewFakeRelative("namespaces/frontend-test"),
							Objects:  vt.ObjectSets(vt.Helper.DeploymentReaderRoleBinding(), vt.Helper.DeploymentReaderRole()),
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type:     node.AbstractNamespace,
					Relative: nomospath.NewFakeRelative("namespaces"),
					Objects:  vt.ObjectSets(vt.Helper.AdminRoleBinding()),
					Children: []*ast.TreeNode{
						{
							Type:     node.Namespace,
							Relative: nomospath.NewFakeRelative("namespaces/frontend"),
							Objects:  vt.ObjectSets(vt.Helper.PodReaderRoleBinding(), vt.Helper.PodReaderRole()),
						},
						{
							Type:     node.Namespace,
							Relative: nomospath.NewFakeRelative("namespaces/frontend-test"),
							Objects:  vt.ObjectSets(vt.Helper.DeploymentReaderRoleBinding(), vt.Helper.DeploymentReaderRole()),
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
