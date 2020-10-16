package status

import (
	"fmt"
	"io"
	"path"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
)

const (
	indentation = "  "
	separator   = "--------------------"
)

// clusterState represents the sync status of all repos on a cluster.
type clusterState struct {
	ref    string
	status string
	error  string
	repos  []*repoState
}

func (c *clusterState) printRows(writer io.Writer) {
	fmt.Fprintf(writer, "%s\n", separator)
	fmt.Fprintf(writer, "%s\n", c.ref)
	if c.status != "" || c.error != "" {
		fmt.Fprintf(writer, "%s\t%s\n", c.status, c.error)
	}
	for _, repo := range c.repos {
		repo.printRows(writer)
	}
}

// repoState represents the sync status of a single repo on a cluster.
type repoState struct {
	scope  string
	git    *v1alpha1.Git
	status string
	commit string
	errors []string
}

func (r *repoState) printRows(writer io.Writer) {
	fmt.Fprintf(writer, "%s\t%s\t\n", r.scope, gitString(r.git))
	fmt.Fprintf(writer, "%s%s\t%s\t\n", indentation, r.status, r.commit)

	for _, err := range r.errors {
		fmt.Fprintf(writer, "%sError:\t%s\t\n", indentation, err)
	}
}

func gitString(git *v1alpha1.Git) string {
	var gitStr string
	if git.Dir != "" {
		gitStr = path.Join(git.Repo, git.Dir)
	} else {
		gitStr = git.Repo
	}

	if git.Revision != "" {
		gitStr = fmt.Sprintf("%s@%s", gitStr, git.Revision)
	} else if git.Branch != "" {
		gitStr = fmt.Sprintf("%s@%s", gitStr, git.Branch)
	} else {
		// Currently git-sync defaults to "master". If/when that changes, then we
		// should update this.
		gitStr = fmt.Sprintf("%s@master", gitStr)
	}

	return gitStr
}

// monoRepoStatus converts the given Git config and mono-repo status into a repoState.
func monoRepoStatus(git *v1alpha1.Git, status v1.RepoStatus) *repoState {
	commit := status.Sync.LatestToken
	if len(commit) == 0 {
		commit = "N/A"
	}

	return &repoState{
		scope:  "<root>",
		git:    git,
		status: getSyncStatus(status),
		commit: commit,
		errors: syncStatusErrors(status),
	}
}
