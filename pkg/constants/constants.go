package constants

import (
	"time"

	"github.com/google/nomos/pkg/api/configsync"
)

const (
	// ControllerNamespace is the Namespace used for Nomos controllers
	ControllerNamespace = "config-management-system"

	// FieldManager is the field manager name for server-side apply.
	FieldManager = configsync.GroupName

	// ConfigSyncPrefix is the prefix for all ConfigSync annotations and labels.
	ConfigSyncPrefix = configsync.GroupName + "/"

	// LifecyclePrefix is the prefix for all lifecycle annotations.
	LifecyclePrefix = "client.lifecycle.config.k8s.io"
)

// API type constants
const (
	// RepoSyncName is the expected name of any RepoSync CR.
	RepoSyncName = "repo-sync"
	// RootSyncName is the expected name of any RootSync CR.
	RootSyncName = "root-sync"
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

// Git secret configmap key names
const (
	// GitSecretConfigKeySSH is the key at which an ssh cert is stored
	GitSecretConfigKeySSH = "ssh"
	// GitSecretConfigKeyCookieFile is the key at which the git cookiefile is stored
	GitSecretConfigKeyCookieFile = "cookie_file"
	// GitSecretConfigKeyToken is the key at which a token's value is stored
	GitSecretConfigKeyToken = "token"
	// GitSecretConfigKeyTokenUsername is the key at which a token's username is stored
	GitSecretConfigKeyTokenUsername = "username"
)

// GitSync period value.
const (
	// DefaultPeriodSecs is the default value in seconds between consecutive syncs.
	DefaultPeriodSecs = 15
)

const (
	// DefaultFilesystemPollingPeriod specifies time between checking the filesystem
	// for udpates to the local Git repository.
	DefaultFilesystemPollingPeriod = 5 * time.Second
)

const (
	// GCPSAAnnotationKey is used to annotate RepoSync/RootSync controller SA when
	// spec.git.auth: gcpserviceaccount is used with Workload Identity enabled on a
	// GKE cluster.
	// https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
	GCPSAAnnotationKey = "iam.gke.io/gcp-service-account"
)
