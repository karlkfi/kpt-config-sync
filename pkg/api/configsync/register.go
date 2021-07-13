package configsync

import "time"

const (
	// GroupName is the name of the group of configsync resources.
	GroupName = "configsync.gke.io"

	// ConfigSyncPrefix is the prefix for all ConfigSync annotations and labels.
	ConfigSyncPrefix = GroupName + "/"

	// FieldManager is the field manager name for server-side apply.
	FieldManager = GroupName

	// ControllerNamespace is the Namespace used for Nomos controllers
	ControllerNamespace = "config-management-system"
)

// API type constants
const (
	// RepoSyncName is the expected name of any RepoSync CR.
	RepoSyncName = "repo-sync"
	// RootSyncName is the expected name of any RootSync CR.
	RootSyncName = "root-sync"
)

const (
	// DefaultPeriodSecs is the default value in seconds between consecutive syncs.
	DefaultPeriodSecs = 15

	// DefaultFilesystemPollingPeriod specifies time between checking the filesystem
	// for udpates to the local Git repository.
	DefaultFilesystemPollingPeriod = 5 * time.Second
)

// Git secret values
const (
	// GitSecretGCENode indicates we will use gcenode for getting the git secret
	GitSecretGCENode = "gcenode"
	// GitSecretSSH indicates the secret is an ssh key
	GitSecretSSH = "ssh"
	// GitSecretCookieFile indicates the secret is a git cookiefile
	GitSecretCookieFile = "cookiefile"
	// GitSecretNone indicates the there is no authn token
	GitSecretNone = "none"
	// GitSecretToken indicates the secret is a username/password
	GitSecretToken = "token"
	// GitSecretGCPServiceAccount indicates the secret is a gcp service account
	// when Workload Identity is enabled on a GKE cluster.
	GitSecretGCPServiceAccount = "gcpserviceaccount"
)
