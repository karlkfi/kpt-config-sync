package backend

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var helper = vt.NewTestHelper()

func allConfigs(c, crd v1.ClusterConfig, pns []v1.NamespaceConfig) *namespaceconfig.AllConfigs {
	ap := &namespaceconfig.AllConfigs{
		ClusterConfig:    &c,
		CRDClusterConfig: &crd,
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
	expect *namespaceconfig.AllConfigs
}

func (tc *OutputVisitorTestcase) Run(t *testing.T) {
	ov := NewOutputVisitor()
	tc.input.Accept(ov)
	actual := ov.AllConfigs()

	if diff := cmp.Diff(tc.expect, actual, resourcequota.ResourceQuantityEqual()); diff != "" {
		t.Errorf("mismatch on expected vs actual: %s", diff)
	}
}

var outputVisitorTestCases = []OutputVisitorTestcase{
	{
		name:  "empty",
		input: helper.EmptyRoot(),
		expect: allConfigs(
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
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.CRDClusterConfigName,
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
		name: "empty cluster configs",
		input: &ast.Root{
			ImportToken: vt.ImportToken,
			LoadTime:    vt.ImportTime,
		},
		expect: allConfigs(
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
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.CRDClusterConfigName,
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
		name:  "cluster configs",
		input: helper.ClusterConfigs(),
		expect: allConfigs(
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
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.CRDClusterConfigName,
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
		name:  "crd cluster configs",
		input: helper.CRDClusterConfig(),
		expect: allConfigs(
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
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.CRDClusterConfigName,
				},
				Spec: v1.ClusterConfigSpec{
					Token:      vt.ImportToken,
					ImportTime: metav1.NewTime(vt.ImportTime),
					Resources: []v1.GenericResources{
						{
							Group: kinds.CustomResourceDefinition().Group,
							Kind:  kinds.CustomResourceDefinition().Kind,
							Versions: []v1.GenericVersionResources{
								{
									Version: kinds.CustomResourceDefinition().Version,
									Objects: []runtime.RawExtension{{Object: helper.CRD()}},
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
		name:  "namespace configs",
		input: helper.NamespaceConfigs(),
		expect: allConfigs(
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
			v1.ClusterConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "ClusterConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: v1.CRDClusterConfigName,
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
}

func TestOutputVisitor(t *testing.T) {
	for _, tc := range outputVisitorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
