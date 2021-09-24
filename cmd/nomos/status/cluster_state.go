package status

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/google/nomos/cmd/nomos/util"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/rootsync"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	commitHashLength = 8
	indent           = "  "
	separator        = "--------------------"
)

// clusterState represents the sync status of all repos on a cluster.
type clusterState struct {
	ref    string
	status string
	error  string
	repos  []*repoState
}

func (c *clusterState) printRows(writer io.Writer) {
	fmt.Fprintln(writer, "")
	fmt.Fprintf(writer, "%s\n", c.ref)
	if c.status != "" || c.error != "" {
		fmt.Fprintf(writer, "%s%s\n", indent, separator)
		fmt.Fprintf(writer, "%s%s\t%s\n", indent, c.status, c.error)
	}
	for _, repo := range c.repos {
		fmt.Fprintf(writer, "%s%s\n", indent, separator)
		repo.printRows(writer)
	}
}

// unavailableCluster returns a clusterState for a cluster that could not be
// reached by a client connection.
func unavailableCluster(ref string) *clusterState {
	return &clusterState{
		ref:    ref,
		status: "N/A",
		error:  "Failed to connect to cluster",
	}
}

// repoState represents the sync status of a single repo on a cluster.
type repoState struct {
	scope     string
	git       v1alpha1.Git
	status    string
	commit    string
	errors    []string
	resources []resourceState
}

func (r *repoState) printRows(writer io.Writer) {
	fmt.Fprintf(writer, "%s%s\t%s\t\n", indent, r.scope, gitString(r.git))
	fmt.Fprintf(writer, "%s%s\t%s\t\n", indent, r.status, r.commit)

	for _, err := range r.errors {
		fmt.Fprintf(writer, "%sError:\t%s\t\n", indent, err)
	}

	if resourceStatus && len(r.resources) > 0 {
		sort.Sort(byNamespaceAndType(r.resources))
		fmt.Fprintf(writer, "%sManaged resources:\n", indent)
		hasSourceHash := r.resources[0].SourceHash != ""
		if !hasSourceHash {
			fmt.Fprintf(writer, "%s\tNAMESPACE\tNAME\tSTATUS\n", indent)
		} else {
			fmt.Fprintf(writer, "%s\tNAMESPACE\tNAME\tSTATUS\tSOURCEHASH\n", indent)
		}
		for _, r := range r.resources {
			if !hasSourceHash {
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", indent, r.Namespace, r.String(), r.Status)
			} else {
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", indent, r.Namespace, r.String(), r.Status, r.SourceHash)
			}
		}
	}
}

func gitString(git v1alpha1.Git) string {
	var gitStr string
	if git.Dir == "" || git.Dir == "." || git.Dir == "/" {
		gitStr = strings.TrimSuffix(git.Repo, "/")
	} else {
		gitStr = strings.TrimSuffix(git.Repo, "/") + "/" + path.Clean(strings.TrimPrefix(git.Dir, "/"))
	}

	if git.Revision != "" && git.Revision != "HEAD" {
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
		commit: commitHash(status.Sync.LatestToken),
		errors: syncStatusErrors(status),
	}
}

// getSyncStatus returns the given RepoStatus formatted as a short summary string.
func getSyncStatus(status v1.RepoStatus) string {
	if hasErrors(status) {
		return util.ErrorMsg
	}
	if len(status.Sync.LatestToken) == 0 {
		return pendingMsg
	}
	if status.Sync.LatestToken == status.Source.Token && len(status.Sync.InProgress) == 0 {
		return syncedMsg
	}
	return pendingMsg
}

// hasErrors returns true if there are any config management errors present in the given RepoStatus.
func hasErrors(status v1.RepoStatus) bool {
	if len(status.Import.Errors) > 0 {
		return true
	}
	for _, syncStatus := range status.Sync.InProgress {
		if len(syncStatus.Errors) > 0 {
			return true
		}
	}
	return false
}

// syncStatusErrors returns all errors reported in the given RepoStatus as a single array.
func syncStatusErrors(status v1.RepoStatus) []string {
	var errs []string
	for _, err := range status.Source.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, err := range status.Import.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, syncStatus := range status.Sync.InProgress {
		for _, err := range syncStatus.Errors {
			errs = append(errs, err.ErrorMessage)
		}
	}

	if getResourceStatus(status.Sync.ResourceConditions) != v1.ResourceStateHealthy {
		errs = append(errs, getResourceStatusErrors(status.Sync.ResourceConditions)...)
	}

	return errs
}

func getResourceStatus(resourceConditions []v1.ResourceCondition) v1.ResourceConditionState {
	resourceStatus := v1.ResourceStateHealthy

	for _, resourceCondition := range resourceConditions {

		if resourceCondition.ResourceState.IsError() {
			return v1.ResourceStateError
		} else if resourceCondition.ResourceState.IsReconciling() {
			resourceStatus = v1.ResourceStateReconciling
		}
	}

	return resourceStatus
}

func getResourceStatusErrors(resourceConditions []v1.ResourceCondition) []string {
	if len(resourceConditions) == 0 {
		return nil
	}

	var syncErrors []string

	for _, resourceCondition := range resourceConditions {
		for _, rcError := range resourceCondition.Errors {
			syncErrors = append(syncErrors, fmt.Sprintf("%v\t%v\tError: %v", resourceCondition.Kind, resourceCondition.NamespacedName, rcError))
		}
		for _, rcReconciling := range resourceCondition.ReconcilingReasons {
			syncErrors = append(syncErrors, fmt.Sprintf("%v\t%v\tReconciling: %v", resourceCondition.Kind, resourceCondition.NamespacedName, rcReconciling))
		}
	}

	return syncErrors
}

// namespaceRepoStatus converts the given RepoSync into a repoState.
func namespaceRepoStatus(rs *v1alpha1.RepoSync, rg *unstructured.Unstructured) *repoState {
	resources, _ := resourceLevelStatus(rg)
	return &repoState{
		scope:     rs.Namespace,
		git:       rs.Spec.Git,
		status:    getRepoStatus(rs),
		commit:    commitHash(rs.Status.Sync.Commit),
		errors:    repoSyncErrors(rs),
		resources: resources,
	}
}

// rootRepoStatus converts the given RootSync into a repoState.
func rootRepoStatus(rs *v1alpha1.RootSync, rg *unstructured.Unstructured) *repoState {
	resources, _ := resourceLevelStatus(rg)
	return &repoState{
		scope:     "<root>",
		git:       rs.Spec.Git,
		status:    getRootStatus(rs),
		commit:    commitHash(rs.Status.Sync.Commit),
		errors:    rootSyncErrors(rs),
		resources: resources,
	}
}

func commitHash(commit string) string {
	if len(commit) == 0 {
		return "N/A"
	} else if len(commit) > commitHashLength {
		commit = commit[:commitHashLength]
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
	if len(status.Source.Errors) > 0 || len(status.Sync.Errors) > 0 || len(status.Rendering.Errors) > 0 {
		return util.ErrorMsg
	}
	if status.Sync.Commit == "" {
		return pendingMsg
	}
	if status.Sync.Commit == status.Source.Commit {
		// if status.Rendering.commit is empty, it is mostly likely a pre-1.9 ACM cluster.
		// In this case, check the sync commit and the source commit.
		if status.Rendering.Commit == "" {
			return syncedMsg
		}
		// Otherwise, check the sync commit and the rendering commit
		if status.Sync.Commit == status.Rendering.Commit {
			return syncedMsg
		}
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
	for _, err := range status.Rendering.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, err := range status.Source.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	for _, err := range status.Sync.Errors {
		errs = append(errs, err.ErrorMessage)
	}
	return errs
}
