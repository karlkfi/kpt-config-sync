package importer

import (
	"testing"

	"github.com/google/nomos/pkg/testing/apiresource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLessVerions(t *testing.T) {
	testCases := []struct {
		name    string
		less    metav1.APIResource
		greater metav1.APIResource
	}{
		{
			name:    "foo1 vs foo10",
			less:    apiresource.Build(apiresource.Roles(), version("foo1")),
			greater: apiresource.Build(apiresource.Roles(), version("foo10")),
		},
		{
			name:    "v11alpha2 vs foo1",
			less:    apiresource.Build(apiresource.Roles(), version("v11alpha2")),
			greater: apiresource.Build(apiresource.Roles(), version("foo1")),
		},
		{
			name:    "v12alpha1 vs v11alpha2",
			less:    apiresource.Build(apiresource.Roles(), version("v12alpha1")),
			greater: apiresource.Build(apiresource.Roles(), version("v11alpha2")),
		},
		{
			name:    "v3beta1 vs v12alpha1",
			less:    apiresource.Build(apiresource.Roles(), version("v3beta1")),
			greater: apiresource.Build(apiresource.Roles(), version("v12alpha1")),
		},
		{
			name:    "v10beta3 vs v3beta1",
			less:    apiresource.Build(apiresource.Roles(), version("v10beta3")),
			greater: apiresource.Build(apiresource.Roles(), version("v3beta1")),
		},
		{
			name:    "v11beta2 vs v10beta3",
			less:    apiresource.Build(apiresource.Roles(), version("v11beta2")),
			greater: apiresource.Build(apiresource.Roles(), version("v10beta3")),
		},
		{
			name:    "v1 vs v11beta2",
			less:    apiresource.Build(apiresource.Roles(), version("v1")),
			greater: apiresource.Build(apiresource.Roles(), version("v11beta2")),
		},
		{
			name:    "v2 vs v1",
			less:    apiresource.Build(apiresource.Roles(), version("v2")),
			greater: apiresource.Build(apiresource.Roles(), version("v1")),
		},
		{
			name:    "v10 vs v2",
			less:    apiresource.Build(apiresource.Roles(), version("v10")),
			greater: apiresource.Build(apiresource.Roles(), version("v2")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !lessVersions(tc.less, tc.greater) {
				t.Fatalf("expected %q to be sorted before %q", tc.less.Version, tc.greater.Version)
			}
		})
	}
}

func TestLarger(t *testing.T) {
	testCases := []struct {
		name    string
		less    string
		greater string
	}{
		{
			name:    "1 vs 2",
			less:    "1",
			greater: "2",
		},
		{
			name:    "2 vs 10",
			less:    "2",
			greater: "10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !larger(tc.greater, tc.less) {
				t.Fatalf("expected %q to be sorted before %q", tc.greater, tc.less)
			}
		})
	}
}
