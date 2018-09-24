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

package backend

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func allPolicies(cp policyhierarchyv1.ClusterPolicy, pns []policyhierarchyv1.PolicyNode) *policyhierarchyv1.AllPolicies {
	ap := &policyhierarchyv1.AllPolicies{
		ClusterPolicy: &cp,
		PolicyNodes:   map[string]policyhierarchyv1.PolicyNode{},
	}
	for _, pn := range pns {
		ap.PolicyNodes[pn.Name] = pn
	}
	return ap
}

type OutputVisitorTestcase struct {
	name   string
	input  *ast.Context
	expect *policyhierarchyv1.AllPolicies
}

func (tc *OutputVisitorTestcase) Run(t *testing.T) {
	ov := NewOutputVisitor()
	tc.input.Accept(ov)
	actual := ov.AllPolicies()
	if !cmp.Equal(actual, tc.expect, vt.ResourceVersionCmp()) {
		t.Errorf("mismatch on expected vs actual: %s",
			cmp.Diff(tc.expect, actual, vt.ResourceVersionCmp()))
	}
}

var outputVisitorTestCases = []OutputVisitorTestcase{
	{
		name:  "empty",
		input: vt.Helper.EmptyContext(),
		expect: allPolicies(
			policyhierarchyv1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: policyhierarchyv1.ClusterPolicyName,
				},
			},
			[]policyhierarchyv1.PolicyNode{},
		),
	},
	{
		name:  "emtpy cluster policies",
		input: &ast.Context{Cluster: &ast.Cluster{}},
		expect: allPolicies(
			policyhierarchyv1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: policyhierarchyv1.ClusterPolicyName,
				},
			},
			[]policyhierarchyv1.PolicyNode{},
		),
	},
	{
		name:  "cluster policies",
		input: vt.Helper.ClusterPolicies(),
		expect: allPolicies(
			policyhierarchyv1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: policyhierarchyv1.ClusterPolicyName,
				},
				Spec: policyhierarchyv1.ClusterPolicySpec{
					ClusterRolesV1:             []rbacv1.ClusterRole{*vt.Helper.NomosAdminClusterRole()},
					ClusterRoleBindingsV1:      []rbacv1.ClusterRoleBinding{*vt.Helper.NomosAdminClusterRoleBinding()},
					PodSecurityPoliciesV1Beta1: []extensionsv1beta1.PodSecurityPolicy{*vt.Helper.NomosPodSecurityPolicy()},
					Resources: []policyhierarchyv1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRole",
							Versions: []policyhierarchyv1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{{Object: vt.Helper.NomosAdminClusterRole()}},
								},
							},
						},
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []policyhierarchyv1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{{Object: vt.Helper.NomosAdminClusterRoleBinding()}},
								},
							},
						},
						{
							Group: "extensions",
							Kind:  "PodSecurityPolicy",
							Versions: []policyhierarchyv1.GenericVersionResources{
								{
									Version: "v1beta1",
									Objects: []runtime.RawExtension{{Object: vt.Helper.NomosPodSecurityPolicy()}},
								},
							},
						},
					},
				},
			},
			[]policyhierarchyv1.PolicyNode{},
		),
	},
	{
		name:  "reserved namespaces",
		input: vt.Helper.ReservedNamespaces(),
		expect: allPolicies(
			policyhierarchyv1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: policyhierarchyv1.ClusterPolicyName,
				},
			},
			[]policyhierarchyv1.PolicyNode{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "testing",
					},
					Spec: policyhierarchyv1.PolicyNodeSpec{
						Type: policyhierarchyv1.ReservedNamespace,
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "more-testing",
					},
					Spec: policyhierarchyv1.PolicyNodeSpec{
						Type: policyhierarchyv1.ReservedNamespace,
					},
				},
			},
		),
	},
	{
		name:  "namespace policies",
		input: vt.Helper.NamespacePolicies(),
		expect: allPolicies(
			policyhierarchyv1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: policyhierarchyv1.ClusterPolicyName,
				},
			},
			[]policyhierarchyv1.PolicyNode{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "acme",
					},
					Spec: policyhierarchyv1.PolicyNodeSpec{
						Type:            policyhierarchyv1.Policyspace,
						Parent:          "",
						RoleBindingsV1:  []rbacv1.RoleBinding{*vt.Helper.AdminRoleBinding()},
						ResourceQuotaV1: vt.Helper.AcmeResourceQuota(),
						Resources: []policyhierarchyv1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []policyhierarchyv1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.AdminRoleBinding()}},
									},
								},
							},
							{
								Group: "",
								Kind:  "ResourceQuota",
								Versions: []policyhierarchyv1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.AcmeResourceQuota()}},
									},
								},
							},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "frontend",
						Labels:      map[string]string{"environment": "prod"},
						Annotations: map[string]string{"has-waffles": "true"},
					},
					Spec: policyhierarchyv1.PolicyNodeSpec{
						Type:            policyhierarchyv1.Namespace,
						Parent:          "acme",
						RoleBindingsV1:  []rbacv1.RoleBinding{*vt.Helper.PodReaderRoleBinding()},
						RolesV1:         []rbacv1.Role{*vt.Helper.PodReaderRole()},
						ResourceQuotaV1: vt.Helper.FrontendResourceQuota(),
						Resources: []policyhierarchyv1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []policyhierarchyv1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.PodReaderRoleBinding()}},
									},
								},
							},
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "Role",
								Versions: []policyhierarchyv1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.PodReaderRole()}},
									},
								},
							},
							{
								Group: "",
								Kind:  "ResourceQuota",
								Versions: []policyhierarchyv1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.FrontendResourceQuota()}},
									},
								},
							},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: policyhierarchyv1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "frontend-test",
						Labels:      map[string]string{"environment": "test"},
						Annotations: map[string]string{"has-waffles": "false"},
					},
					Spec: policyhierarchyv1.PolicyNodeSpec{
						Type:           policyhierarchyv1.Namespace,
						Parent:         "acme",
						RoleBindingsV1: []rbacv1.RoleBinding{*vt.Helper.DeploymentReaderRoleBinding()},
						RolesV1:        []rbacv1.Role{*vt.Helper.DeploymentReaderRole()},
						Resources: []policyhierarchyv1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []policyhierarchyv1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.DeploymentReaderRoleBinding()}},
									},
								},
							},
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "Role",
								Versions: []policyhierarchyv1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.DeploymentReaderRole()}},
									},
								},
							},
						},
					},
				},
			},
		),
	},
}

func TestOutputVisitor(t *testing.T) {
	for _, tc := range outputVisitorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
