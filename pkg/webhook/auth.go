package webhook

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer"
	authenticationv1 "k8s.io/api/authentication/v1"
)

const (
	saGroup          = "system:serviceaccounts"
	saNamespaceGroup = "system:serviceaccounts:" + configmanagement.ControllerNamespace
)

var (
	saImporter = fmt.Sprintf("%s:%s", saNamespaceGroup, importer.Name)
	// TODO(ekitson): Use reconciler.RootSyncName and reconciler.RepoSyncPrefix
	// once webhook configuration is moved to a subpackage and we can avoid an
	// import cycle.
	saRootReconciler  = fmt.Sprintf("%s:%s", saNamespaceGroup, "root-reconciler")
	saNamespacePrefix = fmt.Sprintf("%s:%s-", saNamespaceGroup, "ns-reconciler")
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
	return username == saImporter
}

func isRootReconciler(username string) bool {
	return username == saRootReconciler
}

func canManage(username, manager string) bool {
	if isRootReconciler(username) || manager == "" {
		return true
	}
	username = strings.TrimPrefix(username, saNamespacePrefix)
	return username == manager
}
