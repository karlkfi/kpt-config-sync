package parse

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/vet"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewNamespaceRunner creates a new runnable parser for parsing a Namespace repo.
func NewNamespaceRunner(clusterName, reconcilerName string, scope declared.Scope, fileReader reader.Reader, c client.Client, pollingFrequency time.Duration, resyncPeriod time.Duration, fs FileSource, dc discovery.ServerResourcer, resources *declared.Resources, app applier.Interface, rem remediator.Interface) Parser {
	return &namespace{
		opts: opts{
			clusterName:      clusterName,
			client:           c,
			reconcilerName:   reconcilerName,
			pollingFrequency: pollingFrequency,
			resyncPeriod:     resyncPeriod,
			files:            files{FileSource: fs},
			parser:           NewNamespace(fileReader, dc, scope),
			updater: updater{
				scope:      scope,
				resources:  resources,
				applier:    app,
				remediator: rem,
				cache:      cache{},
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

var _ Parser = &namespace{}

// parseSource implements the Parser interface
func (p *namespace) parseSource(ctx context.Context, state *gitState) ([]core.Object, status.MultiError) {
	filePaths := reader.FilePaths{
		RootDir:   state.policyDir,
		PolicyDir: p.PolicyDir,
		Files:     state.files,
	}
	glog.Infof("Parsing files from git dir: %s", state.policyDir.OSPath())
	start := time.Now()
	cos, err := p.parser.Parse(p.clusterName, true, vet.NoCachedAPIResources, filesystem.NoSyncedCRDs, filePaths)
	metrics.RecordParseErrorAndDuration(ctx, err, start)
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

// setSourceStatus implements the Parser interface
//
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
		return status.APIServerError(err, "failed to get RepoSync for parser")
	}

	if errs == nil {
		// There were no errors getting the git state.
		hasErrs := len(rs.Status.Source.Errors) > 0
		if rs.Status.Source.Commit == state.commit && !hasErrs {
			// We're already synced to this commit and there are no errors to report,
			// so no need to do anything.
			return nil
		}
	}
	cse := status.ToCSE(errs)
	// If we weren't able to get the commit hash, this replaces the value with
	// empty string.
	rs.Status.Source.Commit = state.commit
	// Replace the previous set of errors getting the git state with the current set.
	rs.Status.Source.Errors = cse
	metrics.RecordReconcilerErrors(ctx, "source", len(cse))

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RepoSync source status from parser")
	}
	return nil
}

// setSyncStatus implements the Parser interface
//
// setSyncStatus sets the sync status with a given git state and set of errors.  If errs is empty, all errors
// will be removed from the status.
func (p *namespace) setSyncStatus(ctx context.Context, commit string, errs status.MultiError) error {
	var rs v1alpha1.RepoSync
	if err := p.client.Get(ctx, reposync.ObjectKey(p.scope), &rs); err != nil {
		return status.APIServerError(err, "failed to get RepoSync for parser")
	}

	hasErrs := errs != nil || len(rs.Status.Sync.Errors) > 0
	if rs.Status.Sync.Commit == commit && !hasErrs {
		return nil
	}

	now := metav1.Now()
	cse := status.ToCSE(errs)
	rs.Status.Sync.Commit = commit
	rs.Status.Sync.Errors = cse
	rs.Status.Sync.LastUpdate = now

	metrics.RecordReconcilerErrors(ctx, "sync", len(cse))
	metrics.RecordLastSync(ctx, now.Time)

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RepoSync sync status from parser")
	}
	return nil
}
