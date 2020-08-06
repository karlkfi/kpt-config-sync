package parse

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// opts holds configuration and core functionality required by all
// parse.Runnables.
type opts struct {
	parser filesystem.ConfigParser

	// clusterName is the name of the cluster we're syncing configuration to.
	clusterName string

	// reader knows how to read objects from a Kubernetes cluster.
	reader client.Reader

	// pollingFrequency is how often to re-import configuration from the filesystem.
	//
	// For tests, use zero as it will poll continuously.
	pollingFrequency time.Duration

	// gitDir is the path to the symbolic link of the git repository.
	// git-sync updates the destination of the symbolic link, so we have to check
	// it every time.
	gitDir cmpath.Absolute

	// policyDir is the path to the directory of policies within the git repository.
	policyDir cmpath.Relative

	applier    applier.Applier
	remediator remediator.Remediator
}

// apply marks the passed set of objs as their updated forms, passing them to
// the linked Applier and Remediator.
func (o *opts) apply(ctx context.Context, objs []ast.FileObject) status.MultiError {
	// The Remediator MUST be updated before the applier.
	//
	// The main reason for this is to avoid a race condition where:
	// 1. the first resource of a GVK is added to Git
	// 2. the applier writes that resource to the cluster
	// 3. a user or controller modifies that resource
	// 4. the remediator establishes a new watch for that GVK
	cos := make([]core.Object, len(objs))
	for i, o := range objs {
		cos[i] = o
	}
	err := o.remediator.UpdateDecls(cos)
	if err != nil {
		return status.UndocumentedErrorBuilder.Wrap(err).Build()
	}

	err = o.applier.Apply(ctx, objs)
	if err != nil {
		return status.UndocumentedErrorBuilder.Wrap(err).Build()
	}
	return nil
}

// absPolicyDir returns the absolute path to the policyDir, and the list of all
// observed files in that directory (recursively).
//
// Returns an error if there is some problem resolving symbolic links or in
// listing the files.
func (o *opts) absPolicyDir() (cmpath.Absolute, []cmpath.Absolute, status.MultiError) {
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
