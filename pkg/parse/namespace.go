package parse

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/core"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewNamespaceRunner creates a new runnable parser for parsing a Namespace repo.
func NewNamespaceRunner(scope declared.Scope, fileReader filesystem.Reader, c client.Client, pollingFrequency time.Duration, fs FileSource, dc discovery.ServerResourcer, resources *declared.Resources, app applier.Interface, rem remediator.Interface) Runnable {
	return &namespace{
		opts: opts{
			client:           c,
			pollingFrequency: pollingFrequency,
			files:            files{FileSource: fs},
			parser:           NewNamespaceParser(fileReader, dc, scope),
			updater: updater{
				resources:  resources,
				applier:    app,
				remediator: rem,
			},
			discoveryInterface: dc,
		},
		scope: scope,
	}
}

type namespace struct {
	opts

	// scope is the name of the Namespace this parser is for.
	// It is an error for this parser's repository to contain resources outside of
	// this Namespace.
	scope declared.Scope
}

// Run implements Runnable.
func (p *namespace) Run(ctx context.Context) {
	ticker := time.NewTicker(p.pollingFrequency)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C: // every clock tick
			state, err := p.Read(ctx)
			if err != nil {
				// Invalidate state on error since this could be the result of switching
				// branches or some other operation where inverting the operation would
				// result in repeating a previous state that was checkpointed.
				p.invalidate()
				glog.Error(err)
				continue
			}

			err = p.Parse(ctx, state)
			if err != nil {
				// See comment above.
				p.invalidate()
				glog.Error(err)
			}
		}
	}
}

// Read implements Runnable.
func (p *namespace) Read(ctx context.Context) (*gitState, status.MultiError) {
	state, err := p.readGitState()
	if err != nil {
		// We don't want to bail out immediately as we want to surface this error to
		// the user in the RepoSync's status field.
		if err2 := p.setSourceStatus(ctx, state, err); err2 != nil {
			return nil, status.APIServerError(err2, "setting source status")
		}
		return nil, err
	}
	return &state, nil
}

func (p *namespace) parseSource(state *gitState) ([]core.Object, status.MultiError) {
	glog.Infof("Parsing files from git dir: %s", state.policyDir.OSPath())
	cos, err := p.parser.Parse(p.clusterName, true, filesystem.NoSyncedCRDs, state.policyDir, state.files)
	if err != nil {
		return nil, err
	}

	// Duplicated with root.go.
	e := addAnnotationsAndLabels(cos, p.scope, p.gitContext(), state.commit)
	if e != nil {
		err = status.Append(err, status.InternalErrorf("unable to add annotations and labels: %v", e))
		return nil, err
	}
	return cos, nil
}

// Parse implements Runnable.
func (p *namespace) Parse(ctx context.Context, state *gitState) status.MultiError {
	if p.upToDate(state.policyDir.OSPath()) {
		return nil
	}

	cos, err := p.parseSource(state)
	if err != nil {
		if err2 := p.setSourceStatus(ctx, *state, err); err2 != nil {
			glog.Errorf("Failed to set source status: %v", err2)
		}
		return err
	}

	err = p.update(ctx, cos)
	if err != nil {
		if err2 := p.setSyncStatus(ctx, state.commit, err); err2 != nil {
			glog.Errorf("Failed to set sync status: %v", err2)
		}
		return err
	}

	// Update status to clear errors
	if err2 := p.setSourceStatus(ctx, *state, nil); err2 != nil {
		glog.Errorf("Failed to set source status: %v", err2)
		err = status.Append(err, err2)
	}
	if err2 := p.setSyncStatus(ctx, state.commit, nil); err2 != nil {
		glog.Errorf("Failed to set sync status: %v", err2)
		err = status.Append(err, err2)
	}
	if err != nil {
		return err
	}

	glog.V(4).Infof("Successfully applied all files from git dir: %s", state.policyDir.OSPath())
	// Only checkpoint our state *everything* succeeded, including status update.
	p.checkpoint(state.policyDir.OSPath())
	return nil
}

// setSourceStatus sets the source status with a given git state and set of errors.  If errs is empty, all errors
// will be removed from the status.
func (p *namespace) setSourceStatus(ctx context.Context, state gitState, errs status.MultiError) error {
	// The main idea here is an error-robust way of surfacing to the user that
	// we're having problems reading from our local clone of their git repository.
	// This can happen when Kubernetes does weird things with mounted filesystems,
	// or if an attacker tried to maliciously change the cluster's record of the
	// source of truth.
	var rs v1alpha1.RepoSync
	if err := p.client.Get(ctx, reposync.ObjectKey(p.scope), &rs); err != nil {
		return errors.Wrap(err, "failed to get RepoSync for parser")
	}

	if !status.HasErrors(errs) {
		// There were no errors getting the git state.
		hasErrs := len(rs.Status.Source.Errors) > 0
		if rs.Status.Source.Commit == state.commit && !hasErrs {
			// We're already synced to this commit and there are no errors to report,
			// so no need to do anything.
			return nil
		}
	}
	// If we weren't able to get the commit hash, this replaces the value with
	// empty string.
	rs.Status.Source.Commit = state.commit
	// Replace the previous set of errors getting the git state with the current set.
	rs.Status.Source.Errors = status.ToCSE(errs)

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return errors.Wrap(err, "failed to update RepoSync source status from parser")
	}
	return nil
}

// setSyncStatus sets the sync status with a given git state and set of errors.  If errs is empty, all errors
// will be removed from the status.
func (p *namespace) setSyncStatus(ctx context.Context, commit string, errs status.MultiError) error {
	var rs v1alpha1.RepoSync
	if err := p.client.Get(ctx, reposync.ObjectKey(p.scope), &rs); err != nil {
		return errors.Wrap(err, "failed to get RepoSync for parser")
	}

	hasErrs := status.HasErrors(errs) || len(rs.Status.Sync.Errors) > 0
	if rs.Status.Sync.Commit == commit && !hasErrs {
		return nil
	}

	rs.Status.Sync.Commit = commit
	rs.Status.Sync.Errors = status.ToCSE(errs)
	rs.Status.Sync.LastUpdate = metav1.Now()

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return errors.Wrap(err, "failed to update RepoSync sync status from parser")
	}
	return nil
}
