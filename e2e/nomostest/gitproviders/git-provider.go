package gitproviders

import (
	"github.com/google/nomos/e2e"
)

const (
	// GitUser is the user for all Git providers.
	GitUser = "config-sync-ci-bot"
)

// GitProvider is an interface for the remote Git providers.
type GitProvider interface {
	Type() string
	RemoteURL(port int, name string) string
	SyncURL(name string) string
	CreateRepository(name string) (string, error)
	DeleteRepositories(names ...string) error
}

// NewGitProvider creates a GitProvider for the specific provider type.
func NewGitProvider(provider string) GitProvider {
	switch provider {
	case e2e.Bitbucket:
		return &BitbucketClient{}
	default:
		return &LocalProvider{}
	}
}
