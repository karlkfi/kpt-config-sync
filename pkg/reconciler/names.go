package reconciler

import "fmt"

const (
	// RepoSyncPrefix is the prefix used for all Namespace reconcilers.
	RepoSyncPrefix = "ns-reconciler"
	// RootSyncName is the name of the Root repository reconciler.
	RootSyncName = "root-reconciler"
)

// RepoSyncName returns name in the format ns-reconciler-<namespace>.
func RepoSyncName(namespace string) string {
	return fmt.Sprintf("%s-%s", RepoSyncPrefix, namespace)
}
