package filesystem

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
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

func ToAPIInfo(t *testing.T, resources []*v1.APIResourceList) *utildiscovery.APIInfo {
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

func TestAddScope(t *testing.T) {
	testCases := []struct {
		name      string
		resources []*v1.APIResourceList
		expected  *utildiscovery.APIInfo
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
			root := &ast.Root{}

			// TODO: test w/ CRDs
			p, pErr := NewParserWithFactory(fstesting.NewTestFactory(t), ParserOpt{EnableCRDs: false})
			if pErr != nil {
				t.Fatal(errors.Wrap(pErr, "should have succeeded"))
			}
			rp, rErr := cmpath.NewRoot(cmpath.FromOS("/"))
			if rErr != nil {
				t.Fatal(errors.Wrap(rErr, "should have succeeded"))
			}
			err := p.addScope(root, newFakeDiscoveryClient(tc.resources), rp)
			if err != nil {
				t.Fatal(errors.Wrap(err, "should have succeeded"))
			}

			actual, err := utildiscovery.GetAPIInfo(root)
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

func TestFailAddScope(t *testing.T) {
	testCases := []struct {
		name   string
		client discovery.ServerResourcesInterface
	}{
		{
			name:   "returns error if fail to get server resources",
			client: &failGetServerResourcesDiscoveryClient{},
		},
		{
			name:   "returns invalid server resources",
			client: &invalidServerResourcesDiscoveryClient{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := &ast.Root{}

			// TODO: test w/ CRDs
			p, pErr := NewParserWithFactory(fstesting.NewTestFactory(t), ParserOpt{EnableCRDs: false})
			if pErr != nil {
				t.Fatal(errors.Wrap(pErr, "should have succeeded"))
			}
			rp, rErr := cmpath.NewRoot(cmpath.FromOS("/"))
			if rErr != nil {
				t.Fatal(errors.Wrap(rErr, "should have succeeded"))
			}
			err := p.addScope(root, tc.client, rp)

			if err == nil {
				t.Fatal("Should have failed.")
			}
		})
	}
}
