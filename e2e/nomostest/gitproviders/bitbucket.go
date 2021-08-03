package gitproviders

import (
	"fmt"

	"github.com/google/nomos/e2e"
)

// BitbucketClient is the client that calls the Bitbucket REST APIs.
type BitbucketClient struct{}

// Type returns the provider type.
func (b *BitbucketClient) Type() string {
	return e2e.Bitbucket
}

// RemoteURL returns the Git URL for the Bitbucket repository.
func (b *BitbucketClient) RemoteURL(_ int, repoName string) string {
	return b.SyncURL(repoName)
}

// SyncURL returns a URL for Config Sync to sync from.
func (b *BitbucketClient) SyncURL(repoName string) string {
	return fmt.Sprintf("git@bitbucket.org:%s/%s", GitUser, repoName)
}

// CreateRepository calls the POST API to create a remote repository on Bitbucket.
func (b *BitbucketClient) CreateRepository(name string) (string, error) {
	// TODO(b/194140434): add implementation here
	return "", nil
}

// DeleteRepositories calls the DELETE API to delete all remote repositories on Bitbucket.
// It deletes multiple repos in a single function in order to reuse the access_token.
func (b *BitbucketClient) DeleteRepositories(names ...string) error {
	// TODO(b/194140434): add implementation here
	return nil
}
