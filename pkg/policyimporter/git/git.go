// Package git provides functionality related to Git repos.
package git

import (
	"strings"

	"github.com/pkg/errors"
)

// gitSyncPrefix is prepended to each commit hash by git-sync. See:
// https://github.com/kubernetes/git-sync/blob/v2.0.6/cmd/git-sync/main.go#L278
const gitSyncPrefix string = "rev-"

// CommitHash parses the Git commit hash from the given directory path. The format for the path
// should follow the pattern produced by git-sync: {root}/rev-{hash}/{policyDir}
func CommitHash(dirPath string) (string, error) {
	// First trim the prefix prepended by git-sync.
	i := strings.Index(dirPath, gitSyncPrefix)
	if i < 0 {
		return "", errors.Errorf("directory path %q is missing git-sync prefix %q", dirPath, gitSyncPrefix)
	}
	i += len(gitSyncPrefix)
	hash := dirPath[i:]
	// Now trim the suffix which is the policy dir in the Git repo.
	i = strings.Index(hash, "/")
	if i < 0 {
		return "", errors.Errorf("directory path %q is missing policy directory suffix", dirPath)
	}
	return hash[:i], nil
}
