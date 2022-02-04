package gitproviders

import (
	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/testing"
)

const (
	// GitUser is the user for all Git providers.
	GitUser = "config-sync-ci-bot"
)

// GitProvider is an interface for the remote Git providers.
type GitProvider interface {
	Type() string

	// RemoteURL returns remote URL of the repository.
	// It is used to set the url for the remote origin using `git remote add origin <REMOTE_URL>.
	// For the testing git-server, RemoteURL uses localhost and forwarded port, while SyncURL uses the DNS.
	// For other git providers, RemoteURL should be the same as SyncURL.
	// name refers to the repo name in the format of <NAMESPACE>/<NAME> of RootSync|RepoSync.
	RemoteURL(port int, name string) string

	// SyncURL returns the git repository URL for Config Sync to sync from.
	// name refers to the repo name in the format of <NAMESPACE>/<NAME> of RootSync|RepoSync.
	SyncURL(name string) string
	CreateRepository(name string) (string, error)
	DeleteRepositories(names ...string) error
	DeleteObsoleteRepos() error
}

// NewGitProvider creates a GitProvider for the specific provider type.
func NewGitProvider(t testing.NTB, provider string) GitProvider {
	switch provider {
	case e2e.Bitbucket:
		client, err := newBitbucketClient()
		if err != nil {
			t.Fatal(err)
		}
		return client
	default:
		return &LocalProvider{}
	}
}
