package reconcilermanager

import "fmt"

const (
	// RepoSyncReconcilerPrefix is the prefix used for all Namespace reconcilers.
	RepoSyncReconcilerPrefix = "ns-reconciler"
	// RootSyncName is the name of the Root repository reconciler.
	RootSyncName = "root-reconciler"
	// ManagerName is the name of the controller which creates reconcilers.
	ManagerName = "reconciler-manager"
)

const (
	// SourceFormat is the key used for storing whether a repository is
	// unstructured or in hierarchy mode. Used in many objects related to this
	// behavior.
	SourceFormat = "source-format"

	// ClusterNameKey is the OS env variable and ConfigMap key for the name
	// of the cluster.
	ClusterNameKey = "CLUSTER_NAME"

	// GitSync is the name of the git-sync container in reconciler pods.
	GitSync = "git-sync"

	// Reconciler is a common building block for many resource names associated
	// with reconciling resources.
	Reconciler = "reconciler"
)

const (
	// FilesystemPollingPeriod indicates the time between checking the filesystem
	// for git updates.
	FilesystemPollingPeriod = "FILESYSTEM_POLLING_PERIOD"
)

// RepoSyncName returns name in the format ns-reconciler-<namespace>.
func RepoSyncName(namespace string) string {
	return fmt.Sprintf("%s-%s", RepoSyncReconcilerPrefix, namespace)
}
