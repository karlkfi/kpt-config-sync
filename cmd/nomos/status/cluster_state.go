package status

import (
	"fmt"
	"io"
	"path"

	"github.com/google/nomos/cmd/nomos/util"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/rootsync"
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
	git    v1alpha1.Git
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

func gitString(git v1alpha1.Git) string {
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
func monoRepoStatus(git v1alpha1.Git, status v1.RepoStatus) *repoState {
	return &repoState{
		scope:  "<root>",
		git:    git,
		status: getSyncStatus(status),
		commit: commitOrNA(status.Sync.LatestToken),
		errors: syncStatusErrors(status),
	}
}

// namespaceRepoStatus converts the given RepoSync into a repoState.
func namespaceRepoStatus(rs *v1alpha1.RepoSync) *repoState {
	return &repoState{
		scope:  rs.Namespace,
		git:    rs.Spec.Git,
		status: getRepoStatus(rs),
		commit: commitOrNA(rs.Status.Sync.Commit),
		errors: repoSyncErrors(rs),
	}
}

// rootRepoStatus converts the given RootSync into a repoState.
func rootRepoStatus(rs *v1alpha1.RootSync) *repoState {
	return &repoState{
		scope:  "<root>",
		git:    rs.Spec.Git,
		status: getRootStatus(rs),
		commit: commitOrNA(rs.Status.Sync.Commit),
		errors: rootSyncErrors(rs),
	}
}

func commitOrNA(commit string) string {
	if len(commit) == 0 {
		return "N/A"
	}
	return commit
}

func getRepoStatus(rs *v1alpha1.RepoSync) string {
	if reposync.IsStalled(rs) {
		return util.ErrorMsg
	}
	// TODO(b/168529857): Report reconciling once it is no longer sticky.
	//if reposync.IsReconciling(rs) {
	//	return "DEPLOYING"
	//}
	return multiRepoSyncStatus(rs.Status.SyncStatus)
}

func getRootStatus(rs *v1alpha1.RootSync) string {
	if rootsync.IsStalled(rs) {
		return util.ErrorMsg
	}
	// TODO(b/168529857): Report reconciling once it is no longer sticky.
	//if rootsync.IsReconciling(rs) {
	//	return "DEPLOYING"
	//}
	return multiRepoSyncStatus(rs.Status.SyncStatus)
}

func multiRepoSyncStatus(status v1alpha1.SyncStatus) string {
	if len(status.Source.Errors) > 0 || len(status.Sync.Errors) > 0 {
		return util.ErrorMsg
	}
	if len(status.Sync.Commit) == 0 {
		return pendingMsg
	}
	if status.Sync.Commit == status.Source.Commit {
		return syncedMsg
	}
	return pendingMsg
}

// repoSyncErrors returns all errors reported in the given RepoSync as a single array.
func repoSyncErrors(rs *v1alpha1.RepoSync) []string {
	if reposync.IsStalled(rs) {
		return []string{reposync.StalledMessage(rs)}
	}
	return multiRepoSyncStatusErrors(rs.Status.SyncStatus)
}

// rootSyncErrors returns all errors reported in the given RootSync as a single array.
func rootSyncErrors(rs *v1alpha1.RootSync) []string {
	if rootsync.IsStalled(rs) {
		return []string{rootsync.StalledMessage(rs)}
	}
	return multiRepoSyncStatusErrors(rs.Status.SyncStatus)
}

func multiRepoSyncStatusErrors(status v1alpha1.SyncStatus) []string {
	var errs []string
	for _, err := range status.Source.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, err := range status.Sync.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	return errs
}
