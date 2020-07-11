package webhook

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	authenticationv1 "k8s.io/api/authentication/v1"
)

const (
	saGroup          = "system:serviceaccounts"
	saNamespaceGroup = "system:serviceaccounts:" + configmanagement.ControllerNamespace
)

// isConfigSyncSA returns true if the given UserInfo represents a Config Sync
// service account.
func isConfigSyncSA(userInfo authenticationv1.UserInfo) bool {
	foundSA := false
	foundNS := false

	for _, group := range userInfo.Groups {
		switch group {
		case saGroup:
			foundSA = true
		case saNamespaceGroup:
			foundNS = true
		}
	}
	return foundSA && foundNS
}

// TODO(b/161167923): Remove this check when we turn down the old importer deployment.
func isImporter(username string) bool {
	return username == saNamespaceGroup+":importer"
}
