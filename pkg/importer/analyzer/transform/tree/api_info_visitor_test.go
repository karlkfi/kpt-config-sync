package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

type fakeDiscoveryClient struct {
	resources []*v1.APIResourceList
}

func (c *fakeDiscoveryClient) ServerResources() ([]*v1.APIResourceList, error) {
	return c.resources, nil
}

func (c *fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*v1.APIResourceList, error) {
	return nil, status.InternalError("fakeDiscoveryClient only defines ServerResources()")
}

func (c *fakeDiscoveryClient) ServerPreferredResources() ([]*v1.APIResourceList, error) {
	return nil, status.InternalError("fakeDiscoveryClient only defines ServerResources()")
}

func (c *fakeDiscoveryClient) ServerPreferredNamespacedResources() ([]*v1.APIResourceList, error) {
	return nil, status.InternalError("fakeDiscoveryClient only defines ServerResources()")
}

func newFakeDiscoveryClient(resources []*v1.APIResourceList) discovery.ServerResourcesInterface {
	return &fakeDiscoveryClient{resources: resources}
}

func ToAPIInfo(t *testing.T, resources []*v1.APIResourceList) utildiscovery.Scoper {
	result, err := utildiscovery.NewAPIInfo(resources)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func roleResourceList() []*v1.APIResourceList {
	return []*v1.APIResourceList{
		{
			GroupVersion: kinds.Role().GroupVersion().String(),
			APIResources: []v1.APIResource{
				{
					Group:   kinds.Role().Group,
					Version: kinds.Role().Version,
					Kind:    kinds.Role().Kind,
				},
			},
		},
	}
}

func TestAPIInfoVisitor(t *testing.T) {
	testCases := []struct {
		name      string
		resources []*v1.APIResourceList
		expected  utildiscovery.Scoper
	}{
		{
			name:     "no server resources returns ephemeral resources",
			expected: ToAPIInfo(t, transform.EphemeralResources()),
		},
		{
			name:      "server resource adds to ephemeral resources",
			resources: roleResourceList(),
			expected:  ToAPIInfo(t, append(roleResourceList(), transform.EphemeralResources()...)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newFakeDiscoveryClient(tc.resources)

			root := &ast.Root{}
			root.Accept(tree.NewAPIInfoBuilderVisitor(client, transform.EphemeralResources()))
			actual, err := utildiscovery.GetScoper(root)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expected, actual, cmp.AllowUnexported(utildiscovery.APIInfo{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

type failGetServerResourcesDiscoveryClient struct {
	fakeDiscoveryClient
}

func (c *failGetServerResourcesDiscoveryClient) ServerResources() ([]*v1.APIResourceList, error) {
	return nil, status.InternalError("expected error")
}

type invalidServerResourcesDiscoveryClient struct {
	fakeDiscoveryClient
}

func (c *invalidServerResourcesDiscoveryClient) ServerResources() ([]*v1.APIResourceList, error) {
	return []*v1.APIResourceList{
		{
			GroupVersion: "not/a/valid/groupVersion",
			APIResources: []v1.APIResource{
				{
					Group:   kinds.Role().Group,
					Version: kinds.Role().Version,
					Kind:    kinds.Role().Kind,
				},
			},
		},
	}, nil
}

func TestAPIInfoVisitorError(t *testing.T) {
	testCases := []struct {
		name   string
		client discovery.ServerResourcesInterface
	}{
		{
			name:   "no server resources returns ephemeral resources",
			client: &failGetServerResourcesDiscoveryClient{},
		},
		{
			name:   "server resource adds to ephemeral resources",
			client: &invalidServerResourcesDiscoveryClient{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := &ast.Root{}
			v := tree.NewAPIInfoBuilderVisitor(tc.client, transform.EphemeralResources())
			root.Accept(v)

			if v.Error() == nil {
				t.Fatal("should have failed")
			}
		})
	}
}
