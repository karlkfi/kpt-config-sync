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
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func withName(o runtime.Object, name string) runtime.Object {
	m := o.(metav1.Object)
	m.SetName(name)
	return o
}

var inheritanceVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.CheckingVisitor {
		return NewInheritanceVisitor(
			[]InheritanceSpec{
				InheritanceSpec{
					GroupVersionKind: rbacv1.SchemeGroupVersion.WithKind("RoleBinding"),
				},
				InheritanceSpec{
					GroupVersionKind: corev1.SchemeGroupVersion.WithKind("ResourceQuota"),
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
			Name:  "Inherit policies",
			Input: vt.Helper.AcmeContext(),
			ExpectOutput: &ast.Context{
				Cluster:            vt.Helper.AcmeCluster(),
				ReservedNamespaces: vt.Helper.AcmeReserved(),
				Tree: &ast.TreeNode{
					Type: ast.Policyspace,
					Path: "acme",
					Children: []*ast.TreeNode{
						&ast.TreeNode{
							Type:        ast.Namespace,
							Path:        "acme/frontend",
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
						&ast.TreeNode{
							Type:        ast.Namespace,
							Path:        "acme/frontend-test",
							Labels:      map[string]string{"environment": "test"},
							Annotations: map[string]string{"has-waffles": "false"},
							Objects: vt.ObjectSets(
								vt.Helper.DeployemntReaderRoleBinding(),
								vt.Helper.DeploymentReaderRole(),
								withName(vt.Helper.AdminRoleBinding(), "admin"),
								vt.Helper.AcmeResourceQuota(),
							),
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
