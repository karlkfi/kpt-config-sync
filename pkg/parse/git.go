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

// absPolicyDir returns the absolute path to the policyDir, and the list of all
// observed files in that directory (recursively).
//
// Returns an error if there is some problem resolving symbolic links or in
// listing the files.
func (o *files) absPolicyDir() (cmpath.Absolute, []cmpath.Absolute, status.MultiError) {
	gitDir, err := o.GitDir.EvalSymlinks()
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "evaluating symbolic link to git dir"), o.GitDir.OSPath())
	}
	err = git.CheckClean(gitDir.OSPath())
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "checking that the git repository has no changes"), o.GitDir.OSPath())
	}

	relPolicyDir := gitDir.Join(o.PolicyDir)
	policyDir, err := relPolicyDir.EvalSymlinks()
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "evaluating symbolic link to policy dir"), relPolicyDir.OSPath())
	}

	if policyDir.OSPath() == o.currentPolicyDir {
		glog.V(4).Infof("Git directory is unchanged: %s", policyDir.OSPath())
	} else {
		glog.Infof("Reading updated git dir: %s", policyDir.OSPath())
		o.currentPolicyDir = policyDir.OSPath()
	}

	files, err := git.ListFiles(policyDir)
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "listing files in policy dir"), policyDir.OSPath())
	}
	return policyDir, files, nil
}

// CommitHash returns the current Git commit hash from the Git directory.
func (o *files) CommitHash() (string, error) {
	gitDir, err := o.GitDir.EvalSymlinks()
	if err != nil {
		return "", status.PathWrapError(
			errors.Wrap(err, "evaluating symbolic link to git dir"), o.GitDir.OSPath())
	}
	return git.CommitHash(gitDir.OSPath())
}

func (o *files) gitContext() gitContext {
	return gitContext{
		Repo:   o.GitRepo,
		Branch: o.GitBranch,
		Rev:    o.GitRev,
	}
}
