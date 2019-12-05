// Package git provides functionality related to Git repos.
package git

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// gitSyncPrefix is prepended to each commit hash by git-sync. See:
// https://github.com/kubernetes/git-sync/blob/v2.0.6/cmd/git-sync/main.go#L278
const gitSyncPrefix string = "rev-"

// CommitHash parses the Git commit hash from the given directory path. The format for the path
// should follow the pattern produced by git-sync: /{root}/rev-{hash}
func CommitHash(dirPath string) (string, error) {
	dirName := filepath.Base(dirPath)

	if !strings.HasPrefix(dirName, gitSyncPrefix) {
		return "", errors.Errorf("directory path %q is missing git-sync prefix %q", dirPath, gitSyncPrefix)
	}
	hash := dirName[len(gitSyncPrefix):]
	return hash, nil
}
