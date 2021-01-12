package parse

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// FileSource includes all settings to configure where a Parser reads files from.
type FileSource struct {
	// GitDir is the path to the symbolic link of the git repository.
	GitDir cmpath.Absolute
	// PolicyDir is the path to the directory of policies within the git repository.
	PolicyDir cmpath.Relative
	// GitRepo is the git repo to sync.
	GitRepo string
	// GitBranch is the branch of the git repo to sync.
	GitBranch string
	// GitRev is the revision of the git repo to sync.
	GitRev string
}

// files lists files in a repository and ensures the Git repository hasn't been
// modified from HEAD.
type files struct {
	FileSource

	// currentPolicyDir is the directory (including git commit hash) last seen by the Parser.
	currentPolicyDir string
}

// gitState contains all state read from the mounted Git repo.
type gitState struct {
	// commit is the Git commit hash read from the Git repo.
	commit string
	// policyDir is the absolute path to the policy directory.
	policyDir cmpath.Absolute
	// files is the list of all observed files in the policy directory (recursively).
	files []cmpath.Absolute
}

// readGitState returns the current state read from the mounted Git repo.
//
// Returns an error if there is some problem resolving symbolic links or in
// listing the files. Returns as much information as possible about the state
// of the git repo on error.
func (o *files) readGitState(reconcilerName string) (gitState, status.Error) {
	result := gitState{}

	gitDir, err := o.GitDir.EvalSymlinks()
	if err != nil {
		return result, status.SourceError.Wrap(err).Sprintf("unable to sync repo\n"+
			"Check git-sync logs for more info: kubectl logs -n config-management-system -l reconciler=%s -c git-sync", reconcilerName).Build()
	}

	commit, e := git.CommitHash(gitDir.OSPath())
	if e != nil {
		return result, status.SourceError.Wrap(e).Sprintf("unable to parse commit hash").Build()
	}
	result.commit = commit

	err = git.CheckClean(gitDir.OSPath())
	if err != nil {
		return result, status.PathWrapError(
			errors.Wrap(err, "checking that the git repository has no changes"), o.GitDir.OSPath())
	}

	relPolicyDir := gitDir.Join(o.PolicyDir)
	policyDir, err := relPolicyDir.EvalSymlinks()
	if err != nil {
		return result, status.PathWrapError(
			errors.Wrap(err, "evaluating symbolic link to policy dir"), relPolicyDir.OSPath())
	}
	result.policyDir = policyDir

	if policyDir.OSPath() == o.currentPolicyDir {
		glog.V(4).Infof("Git directory is unchanged: %s", policyDir.OSPath())
	} else {
		glog.Infof("Reading updated git dir: %s", policyDir.OSPath())
		o.currentPolicyDir = policyDir.OSPath()
	}

	files, err := git.ListFiles(policyDir)
	if err != nil {
		return result, status.PathWrapError(
			errors.Wrap(err, "listing files in policy dir"), policyDir.OSPath())
	}
	result.files = files

	return result, nil
}

func (o *files) gitContext() gitContext {
	return gitContext{
		Repo:   o.GitRepo,
		Branch: o.GitBranch,
		Rev:    o.GitRev,
	}
}
