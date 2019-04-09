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

package backend

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var helper = vt.NewTestHelper()

func allPolicies(cp v1.ClusterConfig, pns []v1.NamespaceConfig) *namespaceconfig.AllPolicies {
	ap := &namespaceconfig.AllPolicies{
		ClusterConfig:    &cp,
		NamespaceConfigs: map[string]v1.NamespaceConfig{},
		Syncs:            map[string]v1.Sync{},
	}
	for _, pn := range pns {
		ap.NamespaceConfigs[pn.Name] = pn
	}
	return ap
}

type OutputVisitorTestcase struct {
	name   string
	input  *ast.Root
	expect *namespaceconfig.AllPolicies
}

func (tc *OutputVisitorTestcase) Run(t *testing.T) {
	ov := NewOutputVisitor(false)
	tc.input.Accept(ov)
	actual := ov.AllPolicies()
	if diff := cmp.Diff(tc.expect, actual, resourcequota.ResourceQuantityEqual()); diff != "" {
		t.Errorf("mismatch on expected vs actual: %s", diff)
	}
}

var outputVisitorTestCases = []OutputVisitorTestcase{
	{
		name:  "empty",
		input: helper.EmptyRoot(),
		expect: allPolicies(
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterConfigName,
				},
				Spec: v1.ClusterConfigSpec{
					Token:      vt.ImportToken,
					ImportTime: metav1.NewTime(vt.ImportTime),
				},
			},
			[]v1.NamespaceConfig{},
		),
	},
	{
		name: "empty cluster policies",
		input: &ast.Root{
			ImportToken: vt.ImportToken,
			LoadTime:    vt.ImportTime,
		},
		expect: allPolicies(
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterConfigName,
				},
				Spec: v1.ClusterConfigSpec{
					Token:      vt.ImportToken,
					ImportTime: metav1.NewTime(vt.ImportTime),
				},
			},
			[]v1.NamespaceConfig{},
		),
	},
	{
		name:  "cluster policies",
		input: helper.ClusterConfigs(),
		expect: allPolicies(
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterConfigName,
				},
				Spec: v1.ClusterConfigSpec{
					Token:      vt.ImportToken,
					ImportTime: metav1.NewTime(vt.ImportTime),
					Resources: []v1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRole",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{{Object: helper.NomosAdminClusterRole()}},
								},
							},
						},
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{{Object: helper.NomosAdminClusterRoleBinding()}},
								},
							},
						},
						{
							Group: "policy",
							Kind:  "PodSecurityPolicy",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1beta1",
									Objects: []runtime.RawExtension{{Object: helper.NomosPodSecurityPolicy()}},
								},
							},
						},
					},
				},
			},
			[]v1.NamespaceConfig{},
		),
	},
	{
		name:  "namespace policies",
		input: helper.NamespacePolicies(),
		expect: allPolicies(
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterConfigName,
				},
				Spec: v1.ClusterConfigSpec{
					Token:      vt.ImportToken,
					ImportTime: metav1.NewTime(vt.ImportTime),
				},
			},
			[]v1.NamespaceConfig{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "NamespaceConfig",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "frontend",
						Labels:      map[string]string{"environment": "prod"},
						Annotations: map[string]string{"has-waffles": "true"},
					},
					Spec: v1.NamespaceConfigSpec{
						Token:      vt.ImportToken,
						ImportTime: metav1.NewTime(vt.ImportTime),
						Resources: []v1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: helper.PodReaderRoleBinding()}},
									},
								},
							},
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "Role",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: helper.PodReaderRole()}},
									},
								},
							},
							{
								Group: "",
								Kind:  "ResourceQuota",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: helper.FrontendResourceQuota()}},
									},
								},
							},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "NamespaceConfig",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "frontend-test",
						Labels:      map[string]string{"environment": "test"},
						Annotations: map[string]string{"has-waffles": "false"},
					},
					Spec: v1.NamespaceConfigSpec{
						Token:      vt.ImportToken,
						ImportTime: metav1.NewTime(vt.ImportTime),
						Resources: []v1.GenericResources{
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "RoleBinding",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: helper.DeploymentReaderRoleBinding()}},
									},
								},
							},
							{
								Group: "rbac.authorization.k8s.io",
								Kind:  "Role",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{{Object: helper.DeploymentReaderRole()}},
									},
								},
							},
						},
					},
				},
			},
		),
	},
	{
		name: "syncs",
		input: &ast.Root{
			SystemObjects: []*ast.SystemObject{
				{
					FileObject: ast.FileObject{
						Path: cmpath.FromSlash("<builtin>"),
						Object: &v1.Sync{
							TypeMeta: metav1.TypeMeta{
								APIVersion: v1.SchemeGroupVersion.String(),
								Kind:       "Sync",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "stuff",
							},
						},
					},
				},
			},
		},
		expect: &namespaceconfig.AllPolicies{
			NamespaceConfigs: map[string]v1.NamespaceConfig{},
			ClusterConfig: &v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.ClusterConfigName,
				},
			},
			Syncs: map[string]v1.Sync{
				"stuff": {
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "Sync",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "stuff",
						Finalizers: []string{v1.SyncFinalizer},
					},
				},
			},
		},
	},
}

func TestOutputVisitor(t *testing.T) {
	for _, tc := range outputVisitorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
