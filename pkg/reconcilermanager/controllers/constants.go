package controllers

const (
	repoSyncReconcilerPrefix = "ns-reconciler"
	rootSyncReconcilerName   = "root-reconciler"

	executedOperation = "operation"

	// This is used as a key in a call to r.log.Info
	operationSubjectName = "name"

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
