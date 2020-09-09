package controllers

const (
	repoSyncReconcilerPrefix = "ns-reconciler"
	rootSyncReconcilerName   = "root-reconciler"

	executedOperation = "operation"

	// This is used as a key in a call to r.log.Info
	operationSubjectName = "name"

	gitCredentialVolume = "git-creds"

	// ClusterNameKey is the OS env variable and ConfigMap key for the name
	// of the cluster.
	ClusterNameKey = "CLUSTER_NAME"
)

// Configmaps Suffix.
const (
	SourceFormat = "source-format"

	gitSync = "git-sync"

	reconciler = "reconciler"
)
