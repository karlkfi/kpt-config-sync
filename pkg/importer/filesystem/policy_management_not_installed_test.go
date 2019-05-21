package filesystem

import (
	"testing"

	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/pkg/errors"
	"k8s.io/client-go/discovery"
)

func TestPolicyManagementNotInstalled(t *testing.T) {
	testCases := []struct {
		name       string
		resources  discovery.CachedDiscoveryInterface
		shouldFail bool
	}{
		{
			name:      "success adds no error",
			resources: fstesting.NewFakeCachedDiscoveryClient(fstesting.TestAPIResourceList(fstesting.TestDynamicResources())),
		},
		{
			name:       "fail adds error",
			resources:  fstesting.NewFakeCachedDiscoveryClient(nil),
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := fstesting.NewStubbedClientGetter(t, tc.resources)
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			p := NewParser(
				f,
				ParserOpt{
					Vet:        false,
					Validate:   true,
					EnableCRDs: true,
					Extension:  &NomosVisitorProvider{},
				},
			)
			eb := p.ValidateInstallation()

			if tc.shouldFail {
				if eb == nil {
					t.Fatal("Should have failed.")
				}
			} else {
				if eb != nil {
					t.Fatal("Should not have failed.")
				}
			}
		})
	}
}
