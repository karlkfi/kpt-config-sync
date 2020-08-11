package parse

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// files lists files in a repository and ensures the Git repository hasn't been
// modified from HEAD.
type files struct {
	// gitDir is the path to the symbolic link of the git repository.
	// git-sync updates the destination of the symbolic link, so we have to check
	// it every time.
	gitDir cmpath.Absolute

	// policyDir is the path to the directory of policies within the git repository.
	policyDir cmpath.Relative
}

// absPolicyDir returns the absolute path to the policyDir, and the list of all
// observed files in that directory (recursively).
//
// Returns an error if there is some problem resolving symbolic links or in
// listing the files.
func (o *files) absPolicyDir() (cmpath.Absolute, []cmpath.Absolute, status.MultiError) {
	gitDir, err := o.gitDir.EvalSymlinks()
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "evaluating symbolic link to git dir"), o.gitDir.OSPath())
	}
	err = git.CheckClean(gitDir.OSPath())
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "checking that the git repository has no changes"), o.gitDir.OSPath())
	}

	relPolicyDir := gitDir.Join(o.policyDir)
	policyDir, err := relPolicyDir.EvalSymlinks()
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "evaluating symbolic link to policy dir"), relPolicyDir.OSPath())
	}

	files, err := git.ListFiles(policyDir)
	if err != nil {
		return cmpath.Absolute{}, nil, status.PathWrapError(
			errors.Wrap(err, "listing files in policy dir"), policyDir.OSPath())
	}
	return policyDir, files, nil
}
