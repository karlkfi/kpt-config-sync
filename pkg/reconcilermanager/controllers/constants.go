package controllers

const (
	repoSyncReconcilerPrefix = "ns-reconciler"
	rootSyncReconcilerName   = "root-reconciler"

	rootSyncName = "root-sync"

	executedOperation = "operation"

	gitCredentialVolume = "git-creds"

	gitSecretNone = "none"

	gitSecretGCENode = "gcenode"
)

// Configmaps Suffix.
const (
	importer = "importer"

	SourceFormat = "source-format"

	gitSync = "git-sync"
)

// fsWatcher Container Suffix.
const (
	fsWatcher = "fs-watcher"
)

// reconcilerConfigMapSuffix contains configmaps which are used to create or update
// various configmaps required by Root Reconciler and Namespace Reconciler pods.
var reconcilerConfigMaps = []string{
	importer,     // Used by importer container.
	SourceFormat, // Used by importer container.
	gitSync,      // Used by git-sync container.
}
