package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/vet"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRootRunner creates a new runnable parser for parsing a Root repository.
func NewRootRunner(clusterName string, format filesystem.SourceFormat, fileReader reader.Reader, c client.Client, pollingFrequency time.Duration, fs FileSource, dc discovery.ServerResourcer, resources *declared.Resources, app applier.Interface, rem remediator.Interface) (Runnable, error) {
	opts := opts{
		clusterName:      clusterName,
		client:           c,
		pollingFrequency: pollingFrequency,
		files:            files{FileSource: fs},
		updater: updater{
			scope:      declared.RootReconciler,
			resources:  resources,
			applier:    app,
			remediator: rem,
		},
		discoveryInterface: dc,
	}

	switch format {
	case filesystem.SourceFormatUnstructured:
		opts.parser = filesystem.NewRawParser(fileReader, dc, metav1.NamespaceDefault, declared.RootReconciler)
	case filesystem.SourceFormatHierarchy:
		opts.parser = filesystem.NewParser(fileReader, dc)
	default:
		return nil, errors.Errorf("unknown SourceFormat %q, must be one of %s or %s",
			format, filesystem.SourceFormatUnstructured, filesystem.SourceFormatHierarchy)
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
func (p *root) Read(ctx context.Context) (*gitState, status.MultiError) {
	state, err := p.readGitState()
	if err != nil {
		if err2 := p.setSourceStatus(ctx, state, err); err2 != nil {
			return nil, status.APIServerError(err2, "setting source status")
		}
		return nil, err
	}
	return &state, nil
}

func (p *root) parseSource(ctx context.Context, state *gitState) ([]core.Object, status.MultiError) {
	wantFiles := state.files
	if p.sourceFormat == filesystem.SourceFormatHierarchy {
		// We're using hierarchical mode for the root repository, so ignore files
		// outside of the allowed directories.
		wantFiles = filesystem.FilterHierarchyFiles(state.policyDir, wantFiles)
	}

	filePaths := reader.FilePaths{
		RootDir:   state.policyDir,
		PolicyDir: p.PolicyDir,
		Files:     wantFiles,
	}

	glog.Infof("Parsing files from git dir: %s", state.policyDir.OSPath())
	start := time.Now()
	cos, err := p.parser.Parse(p.clusterName, true, vet.NoCachedAPIResources, filesystem.NoSyncedCRDs, filePaths)
	metrics.RecordParseErrorAndDuration(ctx, err, start)
	if err != nil {
		return nil, err
	}

	// TODO(b/172610552): After the support for Kptfile in a root repo is added, this validator will no longer be needed.
	if err := nonhierarchical.KptfileExistenceValidator.Validate(filesystem.AsFileObjects(cos)); err != nil {
		return nil, err
	}

	// TODO(b/167700852): Inject this function into the Parser.
	if p.sourceFormat == filesystem.SourceFormatUnstructured {
		cos = addImplicitNamespaces(cos)
	}

	// Duplicated with namespace.go.
	e := addAnnotationsAndLabels(cos, declared.RootReconciler, p.gitContext(), state.commit)
	if e != nil {
		err = status.Append(err, status.InternalErrorf("unable to add annotations and labels: %v", e))
		return nil, err
	}
	return cos, nil
}

// Parse implements Runnable.
// TODO(b/167677315): DRY this with the namespace version.
func (p *root) Parse(ctx context.Context, state *gitState) status.MultiError {
	if p.upToDate(state.policyDir.OSPath()) {
		return nil
	}

	cos, err := p.parseSource(ctx, state)
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

func (p *root) setSourceStatus(ctx context.Context, state gitState, errs status.MultiError) error {
	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		return status.APIServerError(err, "failed to get RootSync for parser")
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

	cse := status.ToCSE(errs)
	rs.Status.Source.Commit = state.commit
	rs.Status.Source.Errors = cse

	metrics.RecordReconcilerErrors(ctx, "source", len(cse))

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RootSync source status from parser")
	}
	return nil
}

func (p *root) setSyncStatus(ctx context.Context, commit string, errs status.MultiError) error {
	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		return status.APIServerError(err, "failed to get RootSync for parser")
	}

	hasErrs := status.HasErrors(errs) || len(rs.Status.Sync.Errors) > 0
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
		return status.APIServerError(err, "failed to update RootSync sync status from parser")
	}
	return nil
}

func addImplicitNamespaces(cos []core.Object) []core.Object {
	// namespaces will track the set of Namespaces we expect to exist, and those
	// which actually do.
	namespaces := make(map[string]bool)

	for _, o := range cos {
		if o.GroupVersionKind().GroupKind() == kinds.Namespace().GroupKind() {
			namespaces[o.GetName()] = true
		} else if o.GetNamespace() != "" && !namespaces[o.GetNamespace()] {
			// If unset, this ensures the key exists and is false.
			// Otherwise it has no impact.
			namespaces[o.GetNamespace()] = false
		}
	}

	for ns, isDeclared := range namespaces {
		if isDeclared {
			continue
		}
		cos = append(cos, fake.NamespaceObject(ns,
			core.Name(ns),
			// We do NOT want to delete theses implicit Namespaces when the resources
			// inside them are removed. Note that if the user later declares
			// the Namespace without this annotation, the annotation is removed as
			// expected.
			core.Annotation(lifecycle.Deletion, lifecycle.PreventDeletion),
		))
	}

	return cos
}
