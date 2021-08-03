package gitproviders

import (
	"fmt"

	"github.com/google/nomos/e2e"
)

// LocalProvider refers to the test git-server running on the same test cluster.
type LocalProvider struct{}

// Type returns the provider type.
func (l *LocalProvider) Type() string {
	return e2e.Local
}

// RemoteURL returns the Git URL for connecting to the test git-server.
func (l *LocalProvider) RemoteURL(port int, name string) string {
	return fmt.Sprintf("ssh://git@localhost:%d/git-server/repos/%s", port, name)
}

// SyncURL returns a URL for Config Sync to sync from.
func (l *LocalProvider) SyncURL(name string) string {
	return fmt.Sprintf("git@test-git-server.config-management-system-test:/git-server/repos/%s", name)
}

// CreateRepository returns the local name as the remote repo name.
// It is a no-op for the test git-server because all repos are
// initialized at once in git-server.go.
func (l *LocalProvider) CreateRepository(name string) (string, error) {
	return name, nil
}

// DeleteRepositories is a no-op for the test git-server because the git-server
// will be deleted after the test.
func (l *LocalProvider) DeleteRepositories(...string) error {
	return nil
}
