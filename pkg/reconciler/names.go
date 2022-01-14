package reconciler

import (
	"fmt"

	"github.com/google/nomos/pkg/api/configsync"
)

const (
	// NsReconcilerPrefix is the prefix used for all Namespace reconcilers.
	NsReconcilerPrefix = "ns-reconciler"
	// RootReconcilerPrefix is the prefix usef for all Root reconcilers.
	RootReconcilerPrefix = "root-reconciler"
)

// RootReconcilerName returns the root reconciler's name in the format root-reconciler-<name>.
// If the RootSync name is "root-sync", it returns "root-reconciler" for backward compatibility.
func RootReconcilerName(name string) string {
	if name == configsync.RootSyncName {
		return RootReconcilerPrefix
	}
	return fmt.Sprintf("%s-%s", RootReconcilerPrefix, name)
}

// NsReconcilerName returns the namespace reconciler's name in the format:
// ns-reconciler-<namespace>-<name>-<name-length>
// If the RepoSync name is "repo-sync", it returns "ns-reconciler-<namespace>" for backward compatibility.
func NsReconcilerName(namespace, name string) string {
	if name == configsync.RepoSyncName {
		return fmt.Sprintf("%s-%s", NsReconcilerPrefix, namespace)
	}
	return fmt.Sprintf("%s-%s-%s-%d", NsReconcilerPrefix, namespace, name, len(name))
}
