package webhook

import (
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/reconciler"
	authenticationv1 "k8s.io/api/authentication/v1"
)

const (
	// groups use the plural "serviceaccounts"
	saGroup          = "system:serviceaccounts"
	saNamespaceGroup = saGroup + ":" + configmanagement.ControllerNamespace

	// usernames use the singular "serviceaccount"
	saGroupPrefix          = "system:serviceaccount"
	saNamespaceGroupPrefix = saGroupPrefix + ":" + configmanagement.ControllerNamespace

	saImporter        = saNamespaceGroupPrefix + ":" + importer.Name
	saRootReconciler  = saNamespaceGroupPrefix + ":" + reconciler.RootSyncName
	saNamespacePrefix = saNamespaceGroupPrefix + ":" + reconciler.RepoSyncPrefix + "-"
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
