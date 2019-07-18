package discovery

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
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
				groupVersionKinds: map[schema.GroupVersionKind]bool{
					schema.GroupVersionKind{Group: "com.acme", Version: "v1", Kind: "Anvil"}: true,
				},
				groupKindsNamespaced: map[schema.GroupKind]bool{
					schema.GroupKind{Group: "com.acme", Kind: "Anvil"}: true,
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
				groupVersionKinds: map[schema.GroupVersionKind]bool{
					schema.GroupVersionKind{Group: "com.acme", Version: "v1", Kind: "Anvil"}: true,
				},
				groupKindsNamespaced: map[schema.GroupKind]bool{
					schema.GroupKind{Group: "com.acme", Kind: "Anvil"}: true,
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
				groupVersionKinds: map[schema.GroupVersionKind]bool{
					schema.GroupVersionKind{Group: "com.acme", Version: "v1", Kind: "Anvil"}: true,
					schema.GroupVersionKind{Group: "com.acme", Version: "v2", Kind: "Anvil"}: true,
				},
				groupKindsNamespaced: map[schema.GroupKind]bool{
					schema.GroupKind{Group: "com.acme", Kind: "Anvil"}: true,
				},
			},
		},
		{
			name: "add CRD with empty scope defaults to Namespaced scope",
			crd: v1beta1.CustomResourceDefinition{
				Spec: v1beta1.CustomResourceDefinitionSpec{
					Group: "com.acme",
					Names: v1beta1.CustomResourceDefinitionNames{
						Plural:     "anvils",
						Singular:   "anvil",
						Kind:       "Anvil",
						ShortNames: []string{"av"},
					},
					Versions: []v1beta1.CustomResourceDefinitionVersion{
						{
							Name:   "v1",
							Served: true,
						},
					},
				},
			},
			wantAPIInfo: &APIInfo{
				groupVersionKinds: map[schema.GroupVersionKind]bool{
					schema.GroupVersionKind{Group: "com.acme", Version: "v1", Kind: "Anvil"}: true,
				},
				groupKindsNamespaced: map[schema.GroupKind]bool{
					schema.GroupKind{Group: "com.acme", Kind: "Anvil"}: true,
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
			if diff := cmp.Diff(tc.wantAPIInfo, apiInfo, cmpopts.EquateEmpty(), cmp.AllowUnexported(APIInfo{})); diff != "" {
				t.Errorf("Unexpected APIInfo: %v", diff)
			}
		})
	}
}

func TestAPIInfo_GroupVersionKinds(t *testing.T) {
	testCases := []struct {
		name          string
		resourceLists []*metav1.APIResourceList
		syncs         []*v1.Sync
		expected      map[schema.GroupVersionKind]bool
	}{
		{
			name: "Lists only mentioned gks",
			resourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "rbac/v1",
					APIResources: []metav1.APIResource{
						{
							Kind: "Role",
						},
					},
				},
				{
					GroupVersion: "rbac/v2",
					APIResources: []metav1.APIResource{
						{
							Kind: "Role",
						},
					},
				},
				{
					GroupVersion: "apps/v1",
					APIResources: []metav1.APIResource{
						{
							Kind: "Deployment",
						},
					},
				},
			},
			syncs: []*v1.Sync{
				{
					Spec: v1.SyncSpec{
						Group: "rbac",
						Kind:  "Role",
					},
				},
			},
			expected: map[schema.GroupVersionKind]bool{
				schema.GroupVersionKind{Group: "rbac", Version: "v1", Kind: "Role"}: true,
				schema.GroupVersionKind{Group: "rbac", Version: "v2", Kind: "Role"}: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			api, err := NewAPIInfo(tc.resourceLists)
			if err != nil {
				t.Error(err)
			}

			result := api.GroupVersionKinds(tc.syncs...)
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
