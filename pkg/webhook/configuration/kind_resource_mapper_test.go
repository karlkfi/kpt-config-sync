package configuration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewKindResourceMapper(t *testing.T) {
	testCases := []struct {
		name         string
		apiResources []*metav1.APIResourceList
		want         kindResourceMapper
		wantErr      status.Error
	}{
		{
			name: "no lists",
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{},
		},
		{
			name: "empty list",
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{},
		},
		{
			name: "one type",
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion,
					apiResource("roles", "Role"),
				),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name: "invalid group",
			apiResources: []*metav1.APIResourceList{
				{
					GroupVersion: "///",
					APIResources: []metav1.APIResource{apiResource("roles", "Role")},
				},
			},
			wantErr: status.APIServerError(errors.New("some error"), "message"),
		},
		{
			name: "two kinds",
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion,
					apiResource("roles", "Role"),
					apiResource("rolebindings", "RoleBinding"),
				),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"):        rbacv1.SchemeGroupVersion.WithResource("roles"),
				rbacv1.SchemeGroupVersion.WithKind("RoleBinding"): rbacv1.SchemeGroupVersion.WithResource("rolebindings"),
			},
		},
		{
			name: "two versions",
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion,
					apiResource("roles", "Role"),
				),
				apiResourceList(rbacv1beta1.SchemeGroupVersion,
					apiResource("roles", "Role"),
				),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"):      rbacv1.SchemeGroupVersion.WithResource("roles"),
				rbacv1beta1.SchemeGroupVersion.WithKind("Role"): rbacv1beta1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name: "two groups",
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion,
					apiResource("roles", "Role"),
				),
				apiResourceList(corev1.SchemeGroupVersion,
					apiResource("namespaces", "Namespace"),
				),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"):      rbacv1.SchemeGroupVersion.WithResource("roles"),
				corev1.SchemeGroupVersion.WithKind("Namespace"): corev1.SchemeGroupVersion.WithResource("namespaces"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newKindResourceMapper(tc.apiResources)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got newKindResourceMapper() err = %v, want %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestAddV1CRDs(t *testing.T) {
	testCases := []struct {
		name   string
		mapper kindResourceMapper
		crds   []apiextensionsv1.CustomResourceDefinition
		want   kindResourceMapper
	}{
		{
			name:   "add none",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			want:   map[schema.GroupVersionKind]schema.GroupVersionResource{},
		},
		{
			name:   "add one to empty",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			crds: []apiextensionsv1.CustomResourceDefinition{
				v1CRD(rbacv1.GroupName, map[string]bool{"v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name:   "add two versions to empty",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			crds: []apiextensionsv1.CustomResourceDefinition{
				v1CRD(rbacv1.GroupName, map[string]bool{"v1beta1": true, "v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1beta1.SchemeGroupVersion.WithKind("Role"): rbacv1beta1.SchemeGroupVersion.WithResource("roles"),
				rbacv1.SchemeGroupVersion.WithKind("Role"):      rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name:   "only add served version",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			crds: []apiextensionsv1.CustomResourceDefinition{
				v1CRD(rbacv1.GroupName, map[string]bool{"v1beta1": false, "v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name: "add one to existing",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{
				corev1.SchemeGroupVersion.WithKind("Namespace"): corev1.SchemeGroupVersion.WithResource("namespaces"),
			},
			crds: []apiextensionsv1.CustomResourceDefinition{
				v1CRD(rbacv1.GroupName, map[string]bool{"v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				corev1.SchemeGroupVersion.WithKind("Namespace"): corev1.SchemeGroupVersion.WithResource("namespaces"),
				rbacv1.SchemeGroupVersion.WithKind("Role"):      rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		// This is an edge case we don't need to design around. The test is to
		// document to other code authors what we do, and it's fine if the behavior
		// here changes.
		{
			name: "overwrite existing",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
			crds: []apiextensionsv1.CustomResourceDefinition{
				v1CRD(rbacv1.GroupName, map[string]bool{"v1": true}, "roles2", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles2"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mapper.addV1CRDs(tc.crds)

			if diff := cmp.Diff(tc.mapper, tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func v1CRD(group string, versions map[string]bool, name, kind string) apiextensionsv1.CustomResourceDefinition {
	// versions is a map from version strings to whether they are served by the
	// API Server.
	crd := apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: group,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural: name,
				Kind:   kind,
			},
		},
	}
	for v, served := range versions {
		crd.Spec.Versions = append(crd.Spec.Versions, apiextensionsv1.CustomResourceDefinitionVersion{
			Name:   v,
			Served: served,
		})
	}
	return crd
}

func TestAddV1Beta1CRDs(t *testing.T) {
	testCases := []struct {
		name   string
		mapper kindResourceMapper
		crds   []apiextensionsv1beta1.CustomResourceDefinition
		want   kindResourceMapper
	}{
		{
			name:   "add none",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			want:   map[schema.GroupVersionKind]schema.GroupVersionResource{},
		},
		{
			name:   "add one to empty",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			crds: []apiextensionsv1beta1.CustomResourceDefinition{
				v1Beta1CRD(rbacv1.GroupName, map[string]bool{"v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name:   "add two versions to empty",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			crds: []apiextensionsv1beta1.CustomResourceDefinition{
				v1Beta1CRD(rbacv1.GroupName, map[string]bool{"v1beta1": true, "v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1beta1.SchemeGroupVersion.WithKind("Role"): rbacv1beta1.SchemeGroupVersion.WithResource("roles"),
				rbacv1.SchemeGroupVersion.WithKind("Role"):      rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name:   "only add served version",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			crds: []apiextensionsv1beta1.CustomResourceDefinition{
				v1Beta1CRD(rbacv1.GroupName, map[string]bool{"v1beta1": false, "v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		{
			name: "add one to existing",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{
				corev1.SchemeGroupVersion.WithKind("Namespace"): corev1.SchemeGroupVersion.WithResource("namespaces"),
			},
			crds: []apiextensionsv1beta1.CustomResourceDefinition{
				v1Beta1CRD(rbacv1.GroupName, map[string]bool{"v1": true}, "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				corev1.SchemeGroupVersion.WithKind("Namespace"): corev1.SchemeGroupVersion.WithResource("namespaces"),
				rbacv1.SchemeGroupVersion.WithKind("Role"):      rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
		// This is an edge case we don't need to design around. The test is to
		// document to other code authors what we do, and it's fine if the behavior
		// here changes.
		{
			name: "overwrite existing",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
			crds: []apiextensionsv1beta1.CustomResourceDefinition{
				v1Beta1CRD(rbacv1.GroupName, map[string]bool{"v1": true}, "roles2", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles2"),
			},
		},
		{
			name:   "add deprecated version",
			mapper: map[schema.GroupVersionKind]schema.GroupVersionResource{},
			crds: []apiextensionsv1beta1.CustomResourceDefinition{
				v1beta1CRDDeprecated(rbacv1.GroupName, "v1", "roles", "Role"),
			},
			want: map[schema.GroupVersionKind]schema.GroupVersionResource{
				rbacv1.SchemeGroupVersion.WithKind("Role"): rbacv1.SchemeGroupVersion.WithResource("roles"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mapper.addV1Beta1CRDs(tc.crds)

			if diff := cmp.Diff(tc.mapper, tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func v1Beta1CRD(group string, versions map[string]bool, name, kind string) apiextensionsv1beta1.CustomResourceDefinition {
	// versions is a map from version strings to whether they are served by the
	// API Server.
	crd := apiextensionsv1beta1.CustomResourceDefinition{
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group: group,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: name,
				Kind:   kind,
			},
		},
	}
	for v, served := range versions {
		crd.Spec.Versions = append(crd.Spec.Versions, apiextensionsv1beta1.CustomResourceDefinitionVersion{
			Name:   v,
			Served: served,
		})
	}
	return crd
}

func v1beta1CRDDeprecated(group, version, name, kind string) apiextensionsv1beta1.CustomResourceDefinition {
	//noinspection GoDeprecation
	return apiextensionsv1beta1.CustomResourceDefinition{
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group: group,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: name,
				Kind:   kind,
			},
			Version: version,
		},
	}
}
