package status

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/google/nomos/cmd/nomos/util"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/rootsync"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	commitHashLength = 8
	emptyCommit      = "N/A"
)

// ClusterState represents the sync status of all repos on a cluster.
type ClusterState struct {
	// Ref is the current cluster context
	Ref    string
	status string
	// Error represents the sync errors
	Error   string
	repos   []*repoState
	isMulti *bool
}

func (c *ClusterState) printRows(writer io.Writer) {
	fmt.Fprintln(writer, "")
	fmt.Fprintf(writer, "%s\n", c.Ref)
	if c.status != "" || c.Error != "" {
		fmt.Fprintf(writer, "%s%s\n", util.Indent, util.Separator)
		fmt.Fprintf(writer, "%s%s\t%s\n", util.Indent, c.status, c.Error)
	}
	for _, repo := range c.repos {
		fmt.Fprintf(writer, "%s%s\n", util.Indent, util.Separator)
		repo.printRows(writer)
	}
}

// unavailableCluster returns a ClusterState for a cluster that could not be
// reached by a client connection.
func unavailableCluster(ref string) *ClusterState {
	return &ClusterState{
		Ref:    ref,
		status: "N/A",
		Error:  "Failed to connect to cluster",
	}
}

// repoState represents the sync status of a single repo on a cluster.
type repoState struct {
	scope  string
	git    v1beta1.Git
	status string
	commit string
	errors []string
	// errorSummary summarizes the `errors` field.
	errorSummary *v1beta1.ErrorSummary
	resources    []resourceState
}

func (r *repoState) printRows(writer io.Writer) {
	fmt.Fprintf(writer, "%s%s\t%s\t\n", util.Indent, r.scope, gitString(r.git))
	fmt.Fprintf(writer, "%s%s\t%s\t\n", util.Indent, r.status, r.commit)

	if r.errorSummary != nil && r.errorSummary.TotalCount > 0 {
		if r.errorSummary.Truncated {
			fmt.Fprintf(writer, "%sTotalErrorCount: %d, ErrorTruncated: %v, ErrorCountAfterTruncation: %d\n", util.Indent,
				r.errorSummary.TotalCount, r.errorSummary.Truncated, r.errorSummary.ErrorCountAfterTruncation)
		} else {
			fmt.Fprintf(writer, "%sTotalErrorCount: %d\n", util.Indent, r.errorSummary.TotalCount)
		}
	}

	for _, err := range r.errors {
		fmt.Fprintf(writer, "%sError:\t%s\t\n", util.Indent, err)
	}

	if resourceStatus && len(r.resources) > 0 {
		sort.Sort(byNamespaceAndType(r.resources))
		fmt.Fprintf(writer, "%sManaged resources:\n", util.Indent)
		hasSourceHash := r.resources[0].SourceHash != ""
		if !hasSourceHash {
			fmt.Fprintf(writer, "%s\tNAMESPACE\tNAME\tSTATUS\n", util.Indent)
		} else {
			fmt.Fprintf(writer, "%s\tNAMESPACE\tNAME\tSTATUS\tSOURCEHASH\n", util.Indent)
		}
		for _, r := range r.resources {
			if !hasSourceHash {
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", util.Indent, r.Namespace, r.String(), r.Status)
			} else {
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", util.Indent, r.Namespace, r.String(), r.Status, r.SourceHash)
			}
			if len(r.Conditions) > 0 {
				for _, condition := range r.Conditions {
					fmt.Fprintf(writer, "%s%s%s%s%s\n", util.Indent, util.Indent, util.Indent, util.Indent, condition.Message)
				}
			}
		}
	}
}

func gitString(git v1beta1.Git) string {
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
func monoRepoStatus(git v1beta1.Git, status v1.RepoStatus) *repoState {
	errors := syncStatusErrors(status)
	totalErrorCount := len(errors)

	result := &repoState{
		scope:  "<root>",
		git:    git,
		status: getSyncStatus(status),
		commit: commitHash(status.Sync.LatestToken),
		errors: errors,
	}

	if totalErrorCount > 0 {
		result.errorSummary = &v1beta1.ErrorSummary{
			TotalCount:                totalErrorCount,
			Truncated:                 false,
			ErrorCountAfterTruncation: totalErrorCount,
		}
	}
	return result
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
func namespaceRepoStatus(rs *v1beta1.RepoSync, rg *unstructured.Unstructured, syncingConditionSupported bool) *repoState {
	repostate := &repoState{
		scope:  rs.Namespace,
		git:    rs.Spec.Git,
		commit: emptyCommit,
	}

	stalledCondition := reposync.GetCondition(rs.Status.Conditions, v1beta1.RepoSyncStalled)
	reconcilingCondition := reposync.GetCondition(rs.Status.Conditions, v1beta1.RepoSyncReconciling)
	syncingCondition := reposync.GetCondition(rs.Status.Conditions, v1beta1.RepoSyncSyncing)
	switch {
	case stalledCondition != nil && stalledCondition.Status == metav1.ConditionTrue:
		repostate.status = stalledMsg
		repostate.errors = []string{stalledCondition.Message}
		if stalledCondition.ErrorSummary != nil {
			repostate.errorSummary = stalledCondition.ErrorSummary
		} else {
			repostate.errorSummary = &v1beta1.ErrorSummary{
				TotalCount:                1,
				Truncated:                 false,
				ErrorCountAfterTruncation: 1,
			}
		}
	case reconcilingCondition != nil && reconcilingCondition.Status == metav1.ConditionTrue:
		repostate.status = reconcilingMsg
	case syncingCondition == nil:
		if syncingConditionSupported {
			repostate.status = pendingMsg
		} else {
			// The new status condition is not available in older versions, use the
			// existing logic to compute the status.
			repostate.status = multiRepoSyncStatus(rs.Status.SyncStatus)
			repostate.commit = commitHash(rs.Status.Sync.Commit)
			repostate.errors = repoSyncErrors(rs)
			totalErrorCount := len(repostate.errors)
			if totalErrorCount > 0 {
				repostate.errorSummary = &v1beta1.ErrorSummary{
					TotalCount:                totalErrorCount,
					Truncated:                 false,
					ErrorCountAfterTruncation: totalErrorCount,
				}
			}
			resources, _ := resourceLevelStatus(rg)
			repostate.resources = resources
		}
	case syncingCondition.Status == metav1.ConditionTrue:
		// The sync step is ongoing.
		repostate.status = pendingMsg
		repostate.commit = syncingCondition.Commit
		if syncingCondition.ErrorSummary != nil {
			repostate.errors = toErrorMessage(reposync.Errors(rs, syncingCondition.ErrorSourceRefs))
			repostate.errorSummary = syncingCondition.ErrorSummary
		} else {
			repostate.errors = toErrorMessage(syncingCondition.Errors)
			totalErrorCount := len(repostate.errors)
			if totalErrorCount > 0 {
				repostate.errorSummary = &v1beta1.ErrorSummary{
					TotalCount:                totalErrorCount,
					Truncated:                 false,
					ErrorCountAfterTruncation: totalErrorCount,
				}
			}
		}
	case reposync.ConditionHasNoErrors(*syncingCondition):
		// The sync step finished without any errors.
		repostate.status = syncedMsg
		repostate.commit = syncingCondition.Commit
		resources, _ := resourceLevelStatus(rg)
		repostate.resources = resources
	default:
		// The sync step finished with errors.
		repostate.status = util.ErrorMsg
		repostate.commit = syncingCondition.Commit
		if syncingCondition.ErrorSummary != nil {
			repostate.errors = toErrorMessage(reposync.Errors(rs, syncingCondition.ErrorSourceRefs))
			repostate.errorSummary = syncingCondition.ErrorSummary
		} else {
			repostate.errors = toErrorMessage(syncingCondition.Errors)
			totalErrorCount := len(repostate.errors)
			if totalErrorCount > 0 {
				repostate.errorSummary = &v1beta1.ErrorSummary{
					TotalCount:                totalErrorCount,
					Truncated:                 false,
					ErrorCountAfterTruncation: totalErrorCount,
				}
			}
		}
	}
	return repostate
}

// rootRepoStatus converts the given RootSync into a repoState.
func rootRepoStatus(rs *v1beta1.RootSync, rg *unstructured.Unstructured, syncingConditionSupported bool) *repoState {
	repostate := &repoState{
		scope:  "<root>",
		git:    rs.Spec.Git,
		commit: emptyCommit,
	}
	stalledCondition := rootsync.GetCondition(rs.Status.Conditions, v1beta1.RootSyncStalled)
	reconcilingCondition := rootsync.GetCondition(rs.Status.Conditions, v1beta1.RootSyncReconciling)
	syncingCondition := rootsync.GetCondition(rs.Status.Conditions, v1beta1.RootSyncSyncing)
	switch {
	case stalledCondition != nil && stalledCondition.Status == metav1.ConditionTrue:
		repostate.status = stalledMsg
		repostate.errors = []string{stalledCondition.Message}
		if stalledCondition.ErrorSummary != nil {
			repostate.errorSummary = stalledCondition.ErrorSummary
		} else {
			repostate.errorSummary = &v1beta1.ErrorSummary{
				TotalCount:                1,
				Truncated:                 false,
				ErrorCountAfterTruncation: 1,
			}
		}
	case reconcilingCondition != nil && reconcilingCondition.Status == metav1.ConditionTrue:
		repostate.status = reconcilingMsg
	case syncingCondition == nil:
		if syncingConditionSupported {
			repostate.status = pendingMsg
		} else {
			// The new status condition is not available in older versions, use the
			// existing logic to compute the status.
			repostate.status = multiRepoSyncStatus(rs.Status.SyncStatus)
			repostate.commit = commitHash(rs.Status.Sync.Commit)
			repostate.errors = rootSyncErrors(rs)
			totalErrorCount := len(repostate.errors)
			if totalErrorCount > 0 {
				repostate.errorSummary = &v1beta1.ErrorSummary{
					TotalCount:                totalErrorCount,
					Truncated:                 false,
					ErrorCountAfterTruncation: totalErrorCount,
				}
			}
			resources, _ := resourceLevelStatus(rg)
			repostate.resources = resources
		}
	case syncingCondition.Status == metav1.ConditionTrue:
		// The sync step is ongoing.
		repostate.status = pendingMsg
		repostate.commit = syncingCondition.Commit
		if syncingCondition.ErrorSummary != nil {
			repostate.errors = toErrorMessage(rootsync.Errors(rs, syncingCondition.ErrorSourceRefs))
			repostate.errorSummary = syncingCondition.ErrorSummary
		} else {
			repostate.errors = toErrorMessage(syncingCondition.Errors)
			totalErrorCount := len(repostate.errors)
			if totalErrorCount > 0 {
				repostate.errorSummary = &v1beta1.ErrorSummary{
					TotalCount:                totalErrorCount,
					Truncated:                 false,
					ErrorCountAfterTruncation: totalErrorCount,
				}
			}
		}
	case rootsync.ConditionHasNoErrors(*syncingCondition):
		// The sync step finished without any errors.
		repostate.status = syncedMsg
		repostate.commit = syncingCondition.Commit
		resources, _ := resourceLevelStatus(rg)
		repostate.resources = resources
	default:
		// The sync step finished with errors.
		repostate.status = util.ErrorMsg
		repostate.commit = syncingCondition.Commit
		if syncingCondition.ErrorSummary != nil {
			repostate.errors = toErrorMessage(rootsync.Errors(rs, syncingCondition.ErrorSourceRefs))
			repostate.errorSummary = syncingCondition.ErrorSummary
		} else {
			repostate.errors = toErrorMessage(syncingCondition.Errors)
			totalErrorCount := len(repostate.errors)
			if totalErrorCount > 0 {
				repostate.errorSummary = &v1beta1.ErrorSummary{
					TotalCount:                totalErrorCount,
					Truncated:                 false,
					ErrorCountAfterTruncation: totalErrorCount,
				}
			}
		}
	}
	return repostate
}

func commitHash(commit string) string {
	if len(commit) == 0 {
		return emptyCommit
	} else if len(commit) > commitHashLength {
		commit = commit[:commitHashLength]
	}
	return commit
}

func multiRepoSyncStatus(status v1beta1.SyncStatus) string {
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
func repoSyncErrors(rs *v1beta1.RepoSync) []string {
	if reposync.IsStalled(rs) {
		return []string{reposync.StalledMessage(rs)}
	}
	return multiRepoSyncStatusErrors(rs.Status.SyncStatus)
}

// rootSyncErrors returns all errors reported in the given RootSync as a single array.
func rootSyncErrors(rs *v1beta1.RootSync) []string {
	if rootsync.IsStalled(rs) {
		return []string{rootsync.StalledMessage(rs)}
	}
	return multiRepoSyncStatusErrors(rs.Status.SyncStatus)
}

func multiRepoSyncStatusErrors(status v1beta1.SyncStatus) []string {
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

func toErrorMessage(errs []v1beta1.ConfigSyncError) []string {
	var msg []string
	for _, err := range errs {
		msg = append(msg, err.ErrorMessage)
	}
	return msg
}
