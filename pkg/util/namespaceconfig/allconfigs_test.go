package namespaceconfig_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/testoutput"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func withClusterResources(os ...core.Object) fake.ClusterConfigMutator {
	return func(cc *v1.ClusterConfig) {
		for _, o := range os {
			cc.AddResource(o)
		}
	}
}

func withNamespaceResources(os ...core.Object) core.MetaMutator {
	return func(co core.Object) {
		nsc := co.(*v1.NamespaceConfig)
		for _, o := range os {
			nsc.AddResource(o)
		}
	}
}

func TestNewAllConfigs(t *testing.T) {
	testCases := []struct {
		name        string
		fileObjects []ast.FileObject
		want        *namespaceconfig.AllConfigs
	}{
		{
			name: "empty AllConfigs",
		},
		{
			name: "v1beta1 CRD",
			fileObjects: []ast.FileObject{
				fake.CustomResourceDefinitionV1Beta1(),
			},
			want: &namespaceconfig.AllConfigs{
				CRDClusterConfig: fake.CRDClusterConfigObject(withClusterResources(
					fake.CustomResourceDefinitionV1Beta1Object(),
				)),
				Syncs: testoutput.Syncs(kinds.CustomResourceDefinitionV1Beta1()),
			},
		},
		{
			name: "v1 CRD",
			fileObjects: []ast.FileObject{
				fake.CustomResourceDefinitionV1(),
			},
			want: &namespaceconfig.AllConfigs{
				CRDClusterConfig: fake.CRDClusterConfigObject(withClusterResources(
					fake.CustomResourceDefinitionV1Object(),
				)),
				Syncs: testoutput.Syncs(kinds.CustomResourceDefinitionV1()),
			},
		},
		{
			name: "both v1 and v1beta1 CRDs",
			fileObjects: []ast.FileObject{
				fake.CustomResourceDefinitionV1(),
				fake.CustomResourceDefinitionV1Beta1(),
			},
			want: &namespaceconfig.AllConfigs{
				CRDClusterConfig: fake.CRDClusterConfigObject(withClusterResources(
					fake.CustomResourceDefinitionV1Object(),
					fake.CustomResourceDefinitionV1Beta1Object(),
				)),
				Syncs: testoutput.Syncs(
					kinds.CustomResourceDefinitionV1Beta1(),
					kinds.CustomResourceDefinitionV1(),
				),
			},
		},
		{
			name: "explicit Namespace does not have Deletion lifecycle annotation",
			fileObjects: []ast.FileObject{
				fake.Namespace("namespaces/shipping"),
				fake.Role(core.Name("my-role"), core.Namespace("shipping")),
			},
			want: &namespaceconfig.AllConfigs{
				NamespaceConfigs: map[string]v1.NamespaceConfig{
					"shipping": *fake.NamespaceConfigObject(
						core.Name("shipping"),
						// No Deletion annotation
						withNamespaceResources(
							fake.RoleObject(core.Name("my-role"), core.Namespace("shipping")),
						)),
				},
				Syncs: testoutput.Syncs(kinds.Role()),
			},
		},
		{
			name: "explicit Namespace second does not have Deletion lifecycle annotation",
			fileObjects: []ast.FileObject{
				fake.Role(core.Name("my-role"), core.Namespace("shipping")),
				fake.Namespace("namespaces/shipping"),
			},
			want: &namespaceconfig.AllConfigs{
				NamespaceConfigs: map[string]v1.NamespaceConfig{
					"shipping": *fake.NamespaceConfigObject(
						core.Name("shipping"),
						// No Deletion annotation
						withNamespaceResources(
							fake.RoleObject(core.Name("my-role"), core.Namespace("shipping")),
						)),
				},
				Syncs: testoutput.Syncs(kinds.Role()),
			},
		},
		{
			name: "implicit Namespace has Deletion lifecycle annotation",
			fileObjects: []ast.FileObject{
				fake.Role(core.Name("my-role"), core.Namespace("shipping")),
			},
			want: &namespaceconfig.AllConfigs{
				NamespaceConfigs: map[string]v1.NamespaceConfig{
					"shipping": *fake.NamespaceConfigObject(
						core.Name("shipping"),
						core.Annotation(lifecycle.Deletion, lifecycle.PreventDeletion),
						withNamespaceResources(
							fake.RoleObject(core.Name("my-role"), core.Namespace("shipping")),
						)),
				},
				Syncs: testoutput.Syncs(kinds.Role()),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			want := namespaceconfig.NewAllConfigs("", metav1.Time{}, nil)
			if tc.want != nil {
				if tc.want.ClusterConfig != nil {
					want.ClusterConfig = tc.want.ClusterConfig
				}
				if tc.want.CRDClusterConfig != nil {
					want.CRDClusterConfig = tc.want.CRDClusterConfig
				}
				if tc.want.NamespaceConfigs != nil {
					want.NamespaceConfigs = tc.want.NamespaceConfigs
				}
				if tc.want.Syncs != nil {
					want.Syncs = tc.want.Syncs
				}
			}

			actual := namespaceconfig.NewAllConfigs("", metav1.Time{}, tc.fileObjects)

			if diff := cmp.Diff(want, actual, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
