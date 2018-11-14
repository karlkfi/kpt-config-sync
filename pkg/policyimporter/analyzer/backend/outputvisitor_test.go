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
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func allPolicies(cp v1.ClusterPolicy, pns []v1.PolicyNode) *v1.AllPolicies {
	ap := &v1.AllPolicies{
		ClusterPolicy: &cp,
		PolicyNodes:   map[string]v1.PolicyNode{},
		Syncs:         map[string]v1alpha1.Sync{},
	}
	for _, pn := range pns {
		ap.PolicyNodes[pn.Name] = pn
	}
	return ap
}

type OutputVisitorTestcase struct {
	name   string
	input  *ast.Root
	expect *v1.AllPolicies
}

func (tc *OutputVisitorTestcase) Run(t *testing.T) {
	ov := NewOutputVisitor([]*v1alpha1.Sync{})
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
		input: vt.Helper.EmptyRoot(),
		expect: allPolicies(
			v1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
			},
			[]v1.PolicyNode{},
		),
	},
	{
		name:  "empty cluster policies",
		input: &ast.Root{Cluster: &ast.Cluster{}},
		expect: allPolicies(
			v1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
			},
			[]v1.PolicyNode{},
		),
	},
	{
		name:  "cluster policies",
		input: vt.Helper.ClusterPolicies(),
		expect: allPolicies(
			v1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
				Spec: v1.ClusterPolicySpec{
					Resources: []v1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRole",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{{Object: vt.Helper.NomosAdminClusterRole()}},
								},
							},
						},
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{{Object: vt.Helper.NomosAdminClusterRoleBinding()}},
								},
							},
						},
						{
							Group: "extensions",
							Kind:  "PodSecurityPolicy",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1beta1",
									Objects: []runtime.RawExtension{{Object: vt.Helper.NomosPodSecurityPolicy()}},
								},
							},
						},
					},
				},
			},
			[]v1.PolicyNode{},
		),
	},
	{
		name:  "reserved namespaces",
		input: vt.Helper.ReservedNamespaces(),
		expect: allPolicies(
			v1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
			},
			[]v1.PolicyNode{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "testing",
					},
					Spec: v1.PolicyNodeSpec{
						Type: v1.ReservedNamespace,
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "more-testing",
					},
					Spec: v1.PolicyNodeSpec{
						Type: v1.ReservedNamespace,
					},
				},
			},
		),
	},
	{
		name:  "namespace policies",
		input: vt.Helper.NamespacePolicies(),
		expect: allPolicies(
			v1.ClusterPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterPolicyName,
				},
			},
			[]v1.PolicyNode{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: v1.RootPolicyNodeName,
					},
					Spec: v1.PolicyNodeSpec{
						Type:            v1.Policyspace,
						Parent:          "",
						ResourceQuotaV1: vt.Helper.AcmeResourceQuota(),
						Resources: []v1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.AdminRoleBinding()}},
									},
								},
							},
							{
								Group: "",
								Kind:  "ResourceQuota",
								Versions: []v1.GenericVersionResources{
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
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "frontend",
						Labels:      map[string]string{"environment": "prod"},
						Annotations: map[string]string{"has-waffles": "true"},
					},
					Spec: v1.PolicyNodeSpec{
						Type:            v1.Namespace,
						Parent:          v1.RootPolicyNodeName,
						ResourceQuotaV1: vt.Helper.FrontendResourceQuota(),
						Resources: []v1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.PodReaderRoleBinding()}},
									},
								},
							},
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "Role",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.PodReaderRole()}},
									},
								},
							},
							{
								Group: "",
								Kind:  "ResourceQuota",
								Versions: []v1.GenericVersionResources{
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
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "PolicyNode",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "frontend-test",
						Labels:      map[string]string{"environment": "test"},
						Annotations: map[string]string{"has-waffles": "false"},
					},
					Spec: v1.PolicyNodeSpec{
						Type:   v1.Namespace,
						Parent: v1.RootPolicyNodeName,
						Resources: []v1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: vt.Helper.DeploymentReaderRoleBinding()}},
									},
								},
							},
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "Role",
								Versions: []v1.GenericVersionResources{
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
