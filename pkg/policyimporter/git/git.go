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
// should follow the pattern produced by git-sync: /{root}/rev-{hash}[/{policyDir}]
func CommitHash(dirPath string) (string, error) {
	p := strings.Split(dirPath, "/")
	if len(p) < 3 {
		return "", errors.Errorf("directory path %q is invalid", dirPath)
	}
	h := strings.Split(p[2], gitSyncPrefix)
	if len(h) != 2 {
		return "", errors.Errorf("directory path %q is missing git-sync prefix %q", dirPath, gitSyncPrefix)
	}
	return h[1], nil
}
