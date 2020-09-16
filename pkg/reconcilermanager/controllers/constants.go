package controllers

import (
	"fmt"
)

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

// gceNode specific value.
const (
	// The GCENode* values are interpolated in the prepareGCENodeSnippet function
	gceNodeAskpassImageTag    = "20200616014202"
	gceNodeAskpassSidecarName = "gcenode-askpass-sidecar"
	gceNodeAskpassPort        = 9102
)

var gceNodeAskpassURL = fmt.Sprintf("http://localhost:%v/git_askpass", gceNodeAskpassPort)

const (
	// git-sync container specific environment variables.
	gitSyncName     = "GIT_SYNC_USERNAME"
	gitSyncPassword = "GIT_SYNC_PASSWORD"
)
