package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/validate"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewNamespaceRunner creates a new runnable parser for parsing a Namespace repo.
func NewNamespaceRunner(clusterName, reconcilerName string, scope declared.Scope, fileReader reader.Reader, c client.Client, pollingFrequency time.Duration, resyncPeriod time.Duration, fs FileSource, dc discovery.DiscoveryInterface, resources *declared.Resources, app applier.Interface, rem remediator.Interface) (Parser, error) {
	converter, err := declared.NewValueConverter(dc)
	if err != nil {
		return nil, err
	}

	return &namespace{
		opts: opts{
			clusterName:      clusterName,
			client:           c,
			reconcilerName:   reconcilerName,
			pollingFrequency: pollingFrequency,
			resyncPeriod:     resyncPeriod,
			files:            files{FileSource: fs},
			parser:           filesystem.NewParser(fileReader),
			updater: updater{
				scope:      scope,
				resources:  resources,
				applier:    app,
				remediator: rem,
			},
			discoveryInterface: dc,
			converter:          converter,
		},
		scope: scope,
	}, nil
}

type namespace struct {
	opts

	// scope is the name of the Namespace this parser is for.
	// It is an error for this parser's repository to contain resources outside of
	// this Namespace.
	scope declared.Scope
}

var _ Parser = &namespace{}

func (p *namespace) options() *opts {
	return &(p.opts)
}

// parseSource implements the Parser interface
func (p *namespace) parseSource(ctx context.Context, state gitState) ([]ast.FileObject, status.MultiError) {
	filePaths := reader.FilePaths{
		RootDir:   state.policyDir,
		PolicyDir: p.PolicyDir,
		Files:     state.files,
	}
	crds, err := p.declaredCRDs()
	if err != nil {
		return nil, err
	}
	builder := utildiscovery.ScoperBuilder(p.discoveryInterface)

	glog.Infof("Parsing files from git dir: %s", state.policyDir.OSPath())
	start := time.Now()
	objs, err := p.parser.Parse(filePaths)
	metrics.RecordParseErrorAndDuration(ctx, err, start)
	if err != nil {
		return nil, err
	}

	options := validate.Options{
		ClusterName:  p.clusterName,
		PolicyDir:    p.PolicyDir,
		PreviousCRDs: crds,
		BuildScoper:  builder,
		Converter:    p.converter,
	}
	options = OptionsForScope(options, p.scope)

	objs, err = validate.Unstructured(objs, options)

	metrics.RecordReconcilerNonBlockingErrors(ctx, status.NonBlockingErrors(err))

	if status.HasBlockingErrors(err) {
		return nil, err
	}

	// Duplicated with root.go.
	e := addAnnotationsAndLabels(objs, p.scope, p.gitContext(), state.commit)
	if e != nil {
		err = status.Append(err, status.InternalErrorf("unable to add annotations and labels: %v", e))
		return nil, err
	}
	return objs, err
}

// setSourceStatus implements the Parser interface
//
// setSourceStatus sets the source status with a given git state and set of errors.  If errs is empty, all errors
// will be removed from the status.
func (p *namespace) setSourceStatus(ctx context.Context, newStatus gitStatus) error {
	// The main idea here is an error-robust way of surfacing to the user that
	// we're having problems reading from our local clone of their git repository.
	// This can happen when Kubernetes does weird things with mounted filesystems,
	// or if an attacker tried to maliciously change the cluster's record of the
	// source of truth.
	var rs v1alpha1.RepoSync
	if err := p.client.Get(ctx, reposync.ObjectKey(p.scope), &rs); err != nil {
		return status.APIServerError(err, "failed to get RepoSync for parser")
	}

	cse := status.ToCSE(newStatus.errs)
	// If we weren't able to get the commit hash, this replaces the value with
	// empty string.
	rs.Status.Source.Commit = newStatus.commit
	rs.Status.Source.Git = v1alpha1.GitStatus{
		Repo:     p.GitRepo,
		Revision: p.GitRev,
		Branch:   p.GitBranch,
		Dir:      p.PolicyDir.SlashPath(),
	}
	// Replace the previous set of errors getting the git state with the current set.
	rs.Status.Source.Errors = cse
	rs.Status.Source.LastUpdate = newStatus.lastUpdate

	continueSyncing := true
	if len(cse) > 0 {
		continueSyncing = false
	}
	metrics.RecordPipelineError(ctx, RepoSyncReconcilerType, "source", len(cse))
	reposync.SetSyncing(&rs, continueSyncing, "Source", "Source", newStatus.commit, cse, newStatus.lastUpdate)

	metrics.RecordReconcilerErrors(ctx, "source", len(cse))

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RepoSync source status from parser")
	}
	return nil
}

// setRenderingStatus implements the Parser interface
func (p *namespace) setRenderingStatus(ctx context.Context, oldStatus, newStatus renderingStatus) error {
	if oldStatus.equal(newStatus) {
		return nil
	}

	var rs v1alpha1.RepoSync
	if err := p.client.Get(ctx, reposync.ObjectKey(p.scope), &rs); err != nil {
		return status.APIServerError(err, "failed to get RepoSync for parser")
	}

	if rs.Status.Rendering.Commit != newStatus.commit {
		if newStatus.message == RenderingSkipped {
			metrics.RecordSkipRenderingCount(ctx)
		} else {
			metrics.RecordRenderingCount(ctx)
		}
	}

	cse := status.ToCSE(newStatus.errs)
	rs.Status.Rendering.Commit = newStatus.commit
	rs.Status.Rendering.Git = v1alpha1.GitStatus{
		Repo:     p.GitRepo,
		Revision: p.GitRev,
		Branch:   p.GitBranch,
		Dir:      p.PolicyDir.SlashPath(),
	}
	rs.Status.Rendering.Message = newStatus.message
	rs.Status.Rendering.Errors = cse
	rs.Status.Rendering.LastUpdate = newStatus.lastUpdate

	continueSyncing := true
	if len(cse) > 0 {
		// If rendering errors exist, it should only have one error, so use cse[0] to get the error code.
		metrics.RecordRenderingErrors(ctx, "rendering", len(cse), cse[0].Code)
		continueSyncing = false
	}
	metrics.RecordPipelineError(ctx, RepoSyncReconcilerType, "rendering", len(cse))

	reposync.SetSyncing(&rs, continueSyncing, "Rendering", newStatus.message, newStatus.commit, cse, newStatus.lastUpdate)
	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RepoSync rendering status from parser")
	}
	return nil
}

// setSyncStatus implements the Parser interface
func (p *namespace) setSyncStatus(ctx context.Context, newStatus gitStatus) error {
	var rs v1alpha1.RepoSync
	if err := p.client.Get(ctx, reposync.ObjectKey(p.scope), &rs); err != nil {
		return status.APIServerError(err, "failed to get RepoSync for parser")
	}

	syncErrs := status.ToCSE(newStatus.errs)
	rs.Status.Sync.Commit = newStatus.commit
	rs.Status.Sync.Git = v1alpha1.GitStatus{
		Repo:     p.GitRepo,
		Revision: p.GitRev,
		Branch:   p.GitBranch,
		Dir:      p.PolicyDir.SlashPath(),
	}
	rs.Status.Sync.Errors = syncErrs
	rs.Status.Sync.LastUpdate = newStatus.lastUpdate
	metrics.RecordReconcilerErrors(ctx, "sync", len(syncErrs))
	metrics.RecordPipelineError(ctx, RepoSyncReconcilerType, "sync", len(syncErrs))
	metrics.RecordLastSync(ctx, newStatus.commit, newStatus.lastUpdate.Time)

	var allErrs []v1alpha1.ConfigSyncError
	allErrs = append(allErrs, rs.Status.Source.Errors...)
	allErrs = append(allErrs, syncErrs...)
	if len(allErrs) == 0 {
		rs.Status.LastSyncedCommit = newStatus.commit
	}
	reposync.SetSyncing(&rs, false, "Sync", "Sync Completed", newStatus.commit, allErrs, newStatus.lastUpdate)

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RepoSync sync status from parser")
	}
	return nil
}
