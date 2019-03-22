package filesystem

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

type successServerResourcesInterface struct {
	discovery.ServerResourcesInterface
}

func (f successServerResourcesInterface) ServerResourcesForGroupVersion(gv string) (*v1.APIResourceList, error) {
	return nil, nil
}

type failServerResourcesInterface struct {
	discovery.ServerResourcesInterface
}

func (f failServerResourcesInterface) ServerResourcesForGroupVersion(gv string) (*v1.APIResourceList, error) {
	return nil, vet.InternalError("error")
}

func TestPolicyManagementNotInstalled(t *testing.T) {
	testCases := []struct {
		name       string
		resources  discovery.ServerResourcesInterface
		shouldFail bool
	}{
		{
			name:      "success adds no error",
			resources: successServerResourcesInterface{},
		},
		{
			name:       "fail adds error",
			resources:  failServerResourcesInterface{},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eb := status.ErrorBuilder{}
			validateInstallation(tc.resources, &eb)

			if tc.shouldFail {
				if eb.Build() == nil {
					t.Fatal("Should have failed.")
				}
			} else {
				if eb.Build() != nil {
					t.Fatal("Should not have failed.")
				}
			}
		})
	}
}
