package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRootParser creates a new runnable parser for parsing a Root repository.
func NewRootParser(
	clusterName string,
	format filesystem.SourceFormat,
	fileReader filesystem.Reader,
	c client.Client,
	pollingFrequency time.Duration,
	fs FileSource,
	discoveryInterface discovery.ServerResourcer,
	app applier.Interface,
	rem remediator.Interface,
) (Runnable, error) {
	opts := opts{
		clusterName:      clusterName,
		client:           c,
		pollingFrequency: pollingFrequency,
		files:            files{FileSource: fs},
		updater: updater{
			applier:    app,
			remediator: rem,
		},
	}

	switch format {
	case filesystem.SourceFormatUnstructured:
		opts.parser = filesystem.NewParser(fileReader, discoveryInterface)
	case filesystem.SourceFormatHierarchy:
		opts.parser = filesystem.NewRawParser(fileReader, discoveryInterface)
	default:
		return nil, errors.Errorf("unknown SourceFormat %q", format)
	}
	return &root{opts: opts, sourceFormat: format}, nil
}

type root struct {
	opts

	// sourceFormat defines the structure of the Root repository. Only the Root
	// repository may be SourceFormatHierarchy; all others are implicitly
	// SourceFormatUnstructured.
	sourceFormat filesystem.SourceFormat
}

// Run implements Runnable.
func (p *root) Run(ctx context.Context) {
	ticker := time.NewTicker(p.pollingFrequency)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C: // every clock tick
			state, err := p.Read(ctx)
			if err != nil {
				glog.Error(err)
				continue
			}

			err = p.Parse(ctx, state)
			if err != nil {
				glog.Error(err)
			}
		}
	}
}

// Read implements Runnable.
func (p *root) Read(ctx context.Context) (*gitState, status.MultiError) {
	state, err := p.readGitState()
	if err != nil {
		p.setSourceStatus(ctx, "", err)
		return nil, err
	}
	p.setSourceStatus(ctx, state.commit, err)
	return state, nil
}

// Parse implements Runnable.
// TODO(b/167677315): DRY this with the namespace version.
func (p *root) Parse(ctx context.Context, state *gitState) status.MultiError {
	if p.lastApplied == state.policyDir.OSPath() {
		return nil
	}

	var err status.MultiError
	defer func() {
		p.setSyncStatus(ctx, state.commit, err)
	}()

	wantFiles := state.files
	if p.sourceFormat == filesystem.SourceFormatHierarchy {
		// We're using hierarchical mode for the root repository, so ignore files
		// outside of the allowed directories.
		wantFiles = filesystem.FilterHierarchyFiles(state.policyDir, wantFiles)
	}

	glog.Infof("Parsing files from git dir: %s", state.policyDir.OSPath())
	cos, err := p.parser.Parse(p.clusterName, true, listCrds(ctx, p.client), state.policyDir, wantFiles)
	if err != nil {
		return err
	}

	e := addAnnotationsAndLabels(cos, declared.RootReconciler, p.gitContext(), state.commit)
	if e != nil {
		err = status.Append(err, status.InternalErrorf("unable to add annotations and labels: %v", e))
		return err
	}

	err = status.Append(err, ValidateRepoSyncs.Validate(filesystem.AsFileObjects(cos)))
	if err != nil {
		return err
	}

	err = p.update(ctx, cos)
	if err == nil {
		glog.V(4).Infof("Successfully applied all files from git dir: %s", state.policyDir.OSPath())
		p.lastApplied = state.policyDir.OSPath()
	}
	return err
}

func (p *root) setSourceStatus(ctx context.Context, commit string, errs status.MultiError) {
	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		glog.Errorf("Failed to get RootSync for parser: %v", err)
		return
	}

	hasErrs := (errs != nil && len(errs.Errors()) > 0) || len(rs.Status.Source.Errors) > 0
	if rs.Status.Source.Commit == commit && !hasErrs {
		return
	}

	rs.Status.Source.Commit = commit
	rs.Status.Source.Errors = status.ToCSE(errs)

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		glog.Errorf("Failed to update RootSync status from parser: %v", err)
	}
}

func (p *root) setSyncStatus(ctx context.Context, commit string, errs status.MultiError) {
	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		glog.Errorf("Failed to get RootSync for parser: %v", err)
		return
	}

	hasErrs := (errs != nil && len(errs.Errors()) > 0) || len(rs.Status.Sync.Errors) > 0
	if rs.Status.Sync.Commit == commit && !hasErrs {
		return
	}

	rs.Status.Sync.Commit = commit
	rs.Status.Sync.Errors = status.ToCSE(errs)
	rs.Status.Sync.LastUpdate = metav1.Now()

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		glog.Errorf("Failed to update RootSync status from parser: %v", err)
	}
}
