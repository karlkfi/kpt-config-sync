package filesystem

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/util/discovery"
)

func TestPolicyManagementNotInstalled(t *testing.T) {
	testCases := []struct {
		name       string
		scoper     discovery.Scoper
		shouldFail bool
	}{
		{
			name: "ACM installed",
			scoper: discovery.Scoper{
				kinds.Role().GroupKind():             discovery.NamespaceScope,
				kinds.ConfigManagement().GroupKind(): discovery.ClusterScope,
			},
		},
		{
			name: "ACM not installed",
			scoper: discovery.Scoper{
				kinds.Role().GroupKind(): discovery.NamespaceScope,
			},
			shouldFail: true,
		},
		{
			name: "ACM corrupt installation",
			scoper: discovery.Scoper{
				kinds.Role().GroupKind():             discovery.NamespaceScope,
				kinds.ConfigManagement().GroupKind(): discovery.NamespaceScope,
			},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			eb := validateInstallation(tc.scoper)

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
