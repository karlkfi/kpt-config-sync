package controllers

const (
	repoSyncReconcilerPrefix = "ns-reconciler"
	rootSyncReconcilerName   = "root-reconciler"

	executedOperation = "operation"

	gitCredentialVolume = "git-creds"

	gitSecretNone = "none"

	gitSecretGCENode = "gcenode"
)

// Configmaps Suffix.
const (
	SourceFormat = "source-format"

	gitSync = "git-sync"

	reconciler = "reconciler"
)

// reconcilerConfigMapSuffix contains configmaps which are used to create or update
// various configmaps required by Root Reconciler and Namespace Reconciler pods.
var reconcilerConfigMaps = []string{
	SourceFormat, // Used by reconciler container.
	gitSync,      // Used by git-sync container.
	reconciler,   // Used by reconciler container.
}
