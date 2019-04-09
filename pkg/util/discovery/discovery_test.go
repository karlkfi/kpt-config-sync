package discovery

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddCustomResource(t *testing.T) {
	testCases := []struct {
		name         string
		apiResources []*metav1.APIResourceList
		crd          v1beta1.CustomResourceDefinition
		wantAPIInfo  *APIInfo
	}{
		{
			name: "add CRD with one version to empty APIInfo",
			crd: v1beta1.CustomResourceDefinition{
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group: "com.acme",
					Names: v1beta1.CustomResourceDefinitionNames{
						Plural:     "anvils",
						Singular:   "anvil",
						Kind:       "Anvil",
						ShortNames: []string{"av"},
					},
					Scope: v1beta1.NamespaceScoped,
					Versions: []v1beta1.CustomResourceDefinitionVersion{
						{
							Name:   "v1",
							Served: true,
						},
					},
				},
			},
			wantAPIInfo: &APIInfo{
				groupKindVersions: map[schema.GroupKind][]string{
					schema.GroupKind{Group: "com.acme", Kind: "Anvil"}: {"v1"},
				},
				resources: map[schema.GroupVersionKind]metav1.APIResource{
					schema.GroupVersionKind{Group: "com.acme", Version: "v1", Kind: "Anvil"}: {
						Name:         "anvils",
						SingularName: "anvil",
						Namespaced:   true,
						Group:        "com.acme",
						Kind:         "Anvil",
						ShortNames:   []string{"av"},
						Version:      "v1",
					},
				},
			},
		},
		{
			name: "add CRD with served and unserved versions to empty APIInfo",
			crd: v1beta1.CustomResourceDefinition{
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group: "com.acme",
					Names: v1beta1.CustomResourceDefinitionNames{
						Plural:     "anvils",
						Singular:   "anvil",
						Kind:       "Anvil",
						ShortNames: []string{"av"},
					},
					Scope: v1beta1.NamespaceScoped,
					Versions: []v1beta1.CustomResourceDefinitionVersion{
						{
							Name:   "v1",
							Served: true,
						},
						{
							Name:   "v2",
							Served: false,
						},
					},
				},
			},
			wantAPIInfo: &APIInfo{
				groupKindVersions: map[schema.GroupKind][]string{
					schema.GroupKind{Group: "com.acme", Kind: "Anvil"}: {"v1"},
				},
				resources: map[schema.GroupVersionKind]metav1.APIResource{
					schema.GroupVersionKind{Group: "com.acme", Version: "v1", Kind: "Anvil"}: {
						Name:         "anvils",
						SingularName: "anvil",
						Namespaced:   true,
						Group:        "com.acme",
						Kind:         "Anvil",
						ShortNames:   []string{"av"},
						Version:      "v1",
					},
				},
			},
		},
		{
			name: "add CRD with multiple versions to empty APIInfo",
			crd: v1beta1.CustomResourceDefinition{
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group: "com.acme",
					Names: v1beta1.CustomResourceDefinitionNames{
						Plural:     "anvils",
						Singular:   "anvil",
						Kind:       "Anvil",
						ShortNames: []string{"av"},
					},
					Scope: v1beta1.NamespaceScoped,
					Versions: []v1beta1.CustomResourceDefinitionVersion{
						{
							Name:   "v1",
							Served: true,
						},
						{
							Name:   "v2",
							Served: true,
						},
					},
				},
			},
			wantAPIInfo: &APIInfo{
				groupKindVersions: map[schema.GroupKind][]string{
					schema.GroupKind{Group: "com.acme", Kind: "Anvil"}: {"v1", "v2"},
				},
				resources: map[schema.GroupVersionKind]metav1.APIResource{
					schema.GroupVersionKind{Group: "com.acme", Version: "v1", Kind: "Anvil"}: {
						Name:         "anvils",
						SingularName: "anvil",
						Namespaced:   true,
						Group:        "com.acme",
						Kind:         "Anvil",
						ShortNames:   []string{"av"},
						Version:      "v1",
					},
					schema.GroupVersionKind{Group: "com.acme", Version: "v2", Kind: "Anvil"}: {
						Name:         "anvils",
						SingularName: "anvil",
						Namespaced:   true,
						Group:        "com.acme",
						Kind:         "Anvil",
						ShortNames:   []string{"av"},
						Version:      "v2",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiInfo, err := NewAPIInfo(tc.apiResources)
			if err != nil {
				t.Fatal(errors.Wrap(err, "unexpected error initializing APIInfo"))
			}
			apiInfo.AddCustomResources(&tc.crd)
			if !cmp.Equal(apiInfo, tc.wantAPIInfo, cmpopts.EquateEmpty(), cmp.AllowUnexported(APIInfo{})) {
				t.Errorf("Unexpected APIInfo: %v", cmp.Diff(apiInfo, tc.wantAPIInfo))
			}
		})
	}
}
