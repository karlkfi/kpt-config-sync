package v1alpha1

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
