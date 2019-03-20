package importer

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/apiresource"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	role        = gvr(apiresource.Roles())
	roleBinding = gvr(apiresource.RoleBindings())
	tokenReview = gvr(apiresource.TokenReviews())
)

type errorLister struct{}

func (l errorLister) List(_ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, errors.New("error")
}

type successLister struct {
	objects []unstructured.Unstructured
}

func (l successLister) List(_ metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return &unstructured.UnstructuredList{Items: l.objects}, nil
}

func newSuccessLister(t *testing.T, objects ...ast.FileObject) successLister {
	result := successLister{}

	for _, o := range objects {
		out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o.Object)
		if err != nil {
			t.Fatal(err)
		}
		result.objects = append(result.objects, unstructured.Unstructured{Object: out})
	}

	return result
}

type fakeResourcer struct {
	resources map[schema.GroupVersionResource]Lister
}

func (r fakeResourcer) Resource(gvr schema.GroupVersionResource) Lister {
	if lister := r.resources[gvr]; lister != nil {
		return lister
	}
	return successLister{}
}

func gvr(resource metav1.APIResource) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    resource.Group,
		Version:  resource.Version,
		Resource: resource.Name,
	}
}

func TestResourceLister(t *testing.T) {
	testCases := []struct {
		name           string
		apiResource    metav1.APIResource
		resourcer      Resourcer
		shouldFail     bool
		expectedLength int
	}{
		{
			name:        "returns error if list fails",
			apiResource: apiresource.Roles(),
			resourcer: fakeResourcer{resources: map[schema.GroupVersionResource]Lister{
				role: errorLister{},
			}},
			shouldFail: true,
		},
		{
			name:        "returns empty list if no resources",
			apiResource: apiresource.Roles(),
			resourcer:   fakeResourcer{},
		},
		{
			name:        "returns object list if one resource",
			apiResource: apiresource.Roles(),
			resourcer: fakeResourcer{resources: map[schema.GroupVersionResource]Lister{
				role: newSuccessLister(t, fake.Build(kinds.Role())),
			}},
			expectedLength: 1,
		},
		{
			name:        "returns no objects if not listable",
			apiResource: apiresource.TokenReviews(), // TokenReviews are not listable.
			resourcer: fakeResourcer{resources: map[schema.GroupVersionResource]Lister{
				tokenReview: newSuccessLister(t, fake.Build(kinds.Role())),
			}},
		},
		{
			name:        "returns object list if two resources",
			apiResource: apiresource.Roles(),
			resourcer: fakeResourcer{resources: map[schema.GroupVersionResource]Lister{
				role: newSuccessLister(t, fake.Build(kinds.Role()), fake.Build(kinds.Role())),
			}},
			expectedLength: 2,
		},
		{
			name:        "returns no resources if none of that type",
			apiResource: apiresource.Roles(),
			resourcer: fakeResourcer{resources: map[schema.GroupVersionResource]Lister{
				roleBinding: newSuccessLister(t, fake.Build(kinds.RoleBinding()), fake.Build(kinds.RoleBinding())),
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			resourceLister := NewResourceLister(tc.resourcer)

			eb := &status.ErrorBuilder{}
			actual := resourceLister.List(tc.apiResource, eb)

			err := eb.Build()
			switch {
			case tc.shouldFail && (err == nil):
				t.Fatal("expected error")
			case tc.shouldFail:
				return
			case !tc.shouldFail && (err != nil):
				t.Fatalf(errors.Wrapf(err, "unexpected error").Error())
			}

			if len(actual) != tc.expectedLength {
				t.Fatalf("expected %d resources but got %d", tc.expectedLength, len(actual))
			}
		})
	}
}
