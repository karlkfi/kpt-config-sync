package reconcile

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/util/openapi"
)

func TestClientApplier_Update(t *testing.T) {
	testCases := []struct {
		name      string
		updateErr error
		wantErr   error
	}{
		{
			name:      "Returning no error results in no error",
			updateErr: nil,
			wantErr:   nil,
		},
		{
			name: "Returning not found results in a conflict resolution error",
			updateErr: apierrors.NewNotFound(
				schema.GroupResource{Group: "rbac", Resource: "roles"}, "role"),
			wantErr: syncerclient.ConflictUpdateDoesNotExist(
				errors.New(""), fake.RoleObject()),
		},
		{
			name: "Returning old version conflict results in a conflict resolution error",
			updateErr: apierrors.NewConflict(
				schema.GroupResource{Group: "rbac", Resource: "roles"}, "role",
				errors.New("old version")),
			wantErr: syncerclient.ConflictUpdateOldVersion(
				errors.New(""), fake.RoleObject()),
		},
		{
			name:      "Returning other errors results in a generic ResourceError",
			updateErr: apierrors.NewBadRequest("reason"),
			wantErr:   status.ResourceErrorBuilder.Wrap(errors.New("reason")).Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := &clientApplier{
				discoveryClient: fakeDiscoveryClient{},
				dynamicClient: fakeDynamicClient{
					err: tc.updateErr,
				},
				openAPIResources: nilOpenAPIResources{},
				fights:           newFightDetector(),
				fLogger:          newFightLogger(),
			}

			_, err := a.Update(context.Background(),
				fake.UnstructuredObject(kinds.Role(), core.Name("role")),
				fake.UnstructuredObject(kinds.Role(), core.Name("role")))

			if !errors.Is(tc.wantErr, err) {
				t.Errorf("got Update() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

type fakeDiscoveryClient struct {
	discovery.DiscoveryInterface
}

func (f fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	return &metav1.APIResourceList{
		GroupVersion: groupVersion,
		APIResources: []metav1.APIResource{
			{Kind: "Role", Name: "roles"},
		},
	}, nil
}

var _ discovery.DiscoveryInterface = fakeDiscoveryClient{}

type fakeDynamicClient struct {
	dynamic.ResourceInterface
	err error
}

func (c fakeDynamicClient) Resource(schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return c
}

func (c fakeDynamicClient) Namespace(string) dynamic.ResourceInterface {
	return c
}

func (c fakeDynamicClient) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*unstructured.Unstructured, error) {
	return nil, c.err
}

type nilOpenAPIResources struct{}

var _ openapi.Resources = nilOpenAPIResources{}

func (n nilOpenAPIResources) LookupResource(schema.GroupVersionKind) proto.Schema {
	return nil
}
