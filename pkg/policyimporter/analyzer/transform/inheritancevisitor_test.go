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
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors/seltest"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func withName(o runtime.Object, name string) runtime.Object {
	m := o.(metav1.Object)
	m.SetName(name)
	return o
}

var inheritanceVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.CheckingVisitor {
		return NewInheritanceVisitor(
			map[schema.GroupKind]*InheritanceSpec{
				rbacv1.SchemeGroupVersion.WithKind("RoleBinding").GroupKind(): {
					Mode: "inherit",
				},
				corev1.SchemeGroupVersion.WithKind("ResourceQuota").GroupKind(): {
					Mode: "inherit",
				},
			},
		)
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:         "preserve cluster policies",
			Input:        vt.Helper.ClusterPolicies(),
			ExpectOutput: vt.Helper.ClusterPolicies(),
		},
		{
			Name:         "preserve reserved namespaces",
			Input:        vt.Helper.ReservedNamespaces(),
			ExpectOutput: vt.Helper.ReservedNamespaces(),
		},
		{
			Name:  "inherit policies",
			Input: vt.Helper.AcmeRoot(),
			ExpectOutput: &ast.Root{
				Cluster:            vt.Helper.AcmeCluster(),
				ReservedNamespaces: vt.Helper.AcmeReserved(),
				Tree: &ast.TreeNode{
					Type: ast.AbstractNamespace,
					Path: "namespaces",
					Children: []*ast.TreeNode{
						{
							Type:        ast.Namespace,
							Path:        "namespaces/frontend",
							Labels:      map[string]string{"environment": "prod"},
							Annotations: map[string]string{"has-waffles": "true"},
							Objects: vt.ObjectSets(
								vt.Helper.PodReaderRoleBinding(),
								vt.Helper.PodReaderRole(),
								vt.Helper.FrontendResourceQuota(),
								withName(vt.Helper.AdminRoleBinding(), "admin"),
								vt.Helper.AcmeResourceQuota(),
							),
						},
						{
							Type:        ast.Namespace,
							Path:        "namespaces/frontend-test",
							Labels:      map[string]string{"environment": "test"},
							Annotations: map[string]string{"has-waffles": "false"},
							Objects: vt.ObjectSets(
								vt.Helper.DeploymentReaderRoleBinding(),
								vt.Helper.DeploymentReaderRole(),
								withName(vt.Helper.AdminRoleBinding(), "admin"),
								vt.Helper.AcmeResourceQuota(),
							),
						},
					},
				},
			},
		},
		{
			Name: "inherit filtered by NamespaceSelector",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.AbstractNamespace,
					Objects: vt.ObjectSets(
						withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
					),
					Children: []*ast.TreeNode{
						{
							Type:   ast.Namespace,
							Path:   "namespaces/frontend",
							Labels: map[string]string{"env": "prod"},
						},
						{
							Type:   ast.Namespace,
							Path:   "namespaces/frontend-test",
							Labels: map[string]string{"env": "test"},
						},
					},
				},
			},
			ExpectOutput: &ast.Root{
				Tree: &ast.TreeNode{
					Type: ast.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Type:   ast.Namespace,
							Path:   "namespaces/frontend",
							Labels: map[string]string{"env": "prod"},
							Objects: vt.ObjectSets(
								withNamespaceSelector(vt.Helper.AdminRoleBinding(), toJSON(seltest.ProdNamespaceSelector)),
							),
						},
						{
							Type:   ast.Namespace,
							Path:   "namespaces/frontend-test",
							Labels: map[string]string{"env": "test"},
						},
					},
				},
			},
		},
	},
}

func TestInheritanceVisitor(t *testing.T) {
	inheritanceVisitorTestcases.Run(t)
}
