package cloner

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/testing/apiresource"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type errorAPIResourcesLister struct{}

func (l errorAPIResourcesLister) ServerResources() ([]*metav1.APIResourceList, error) {
	return nil, errors.New("error")
}

type successAPIResourcesLister struct {
	resources []*metav1.APIResourceList
}

func (l successAPIResourcesLister) ServerResources() ([]*metav1.APIResourceList, error) {
	return l.resources, nil
}

func group(group string) apiresource.Opt {
	return func(r *metav1.APIResource) {
		r.Group = group
	}
}

func version(version string) apiresource.Opt {
	return func(r *metav1.APIResource) {
		r.Version = version
	}
}

func TestScanAPIResources(t *testing.T) {
	testCases := []struct {
		name              string
		lister            APIResourcesLister
		expected          []metav1.APIResource
		shouldReturnError bool
	}{
		{
			name:              "error retrieving APIResourceLists",
			lister:            errorAPIResourcesLister{},
			shouldReturnError: true,
		},
		{
			name:   "zero APIResourceLists",
			lister: successAPIResourcesLister{},
		},
		{
			name:   "one empty APIResourceList",
			lister: successAPIResourcesLister{resources: apiresource.Lists()},
		},
		{
			name: "one APIResourceList with resource",
			lister: successAPIResourcesLister{
				resources: apiresource.Lists(apiresource.Roles()),
			},
			expected: []metav1.APIResource{apiresource.Roles()},
		},
		{
			name: "error retrieving APIResourceList with invalid group",
			lister: successAPIResourcesLister{
				resources: apiresource.Lists(apiresource.Build(apiresource.Roles(), group("//"))),
			},
			shouldReturnError: true,
		},
		{
			name: "one APIResourceList with resource with multiple verbs",
			lister: successAPIResourcesLister{
				resources: apiresource.Lists(apiresource.Roles()),
			},
			expected: []metav1.APIResource{apiresource.Roles()},
		},
		{
			name: "one APIResourceList with two resources",
			lister: successAPIResourcesLister{
				resources: apiresource.Lists(apiresource.Roles(), apiresource.RoleBindings()),
			},
			expected: []metav1.APIResource{apiresource.Roles(), apiresource.RoleBindings()},
		},
		{
			name: "two APIResourceLists with one resource each",
			lister: successAPIResourcesLister{
				resources: apiresource.Lists(apiresource.Roles(), apiresource.Clusters()),
			},
			expected: []metav1.APIResource{apiresource.Roles(), apiresource.Clusters()},
		},
		{
			name: "two APIResources different versions",
			lister: successAPIResourcesLister{
				resources: apiresource.Lists(
					apiresource.Build(apiresource.Roles(), version("v1")),
					apiresource.Build(apiresource.Roles(), version("v2"))),
			},
			expected: []metav1.APIResource{apiresource.Build(apiresource.Roles(), version("v2"))},
		},
		{
			// This test verifies that ListResources still returns what it can so the user may
			// choose to continue despite errors.
			name: "two APIResourceLists with one resource each, one with invalid group",
			lister: successAPIResourcesLister{
				resources: apiresource.Lists(
					apiresource.Build(apiresource.Roles(), group("//")),
					apiresource.Clusters()),
			},
			shouldReturnError: true,
			expected:          []metav1.APIResource{apiresource.Clusters()},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ListResources(tc.lister)

			switch {
			case tc.shouldReturnError && (err == nil):
				t.Fatal("expected error")
			case !tc.shouldReturnError && (err != nil):
				t.Fatal(err)
			}

			sort.Slice(tc.expected, func(i, j int) bool {
				return tc.expected[i].String() < tc.expected[j].String()
			})

			sort.Slice(actual, func(i, j int) bool {
				return actual[i].String() < actual[j].String()
			})

			if diff := cmp.Diff(tc.expected, actual, cmpopts.IgnoreFields(metav1.APIResource{}, "Verbs")); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
