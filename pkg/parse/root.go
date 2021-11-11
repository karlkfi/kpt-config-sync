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
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/validate"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRootRunner creates a new runnable parser for parsing a Root repository.
func NewRootRunner(clusterName, reconcilerName string, format filesystem.SourceFormat, fileReader reader.Reader, c client.Client, pollingFrequency time.Duration, resyncPeriod time.Duration, fs FileSource, dc discovery.DiscoveryInterface, resources *declared.Resources, app applier.Interface, rem remediator.Interface) (Parser, error) {
	converter, err := declared.NewValueConverter(dc)
	if err != nil {
		return nil, err
	}

	opts := opts{
		clusterName:      clusterName,
		reconcilerName:   reconcilerName,
		client:           c,
		pollingFrequency: pollingFrequency,
		resyncPeriod:     resyncPeriod,
		files:            files{FileSource: fs},
		parser:           filesystem.NewParser(fileReader),
		updater: updater{
			scope:      declared.RootReconciler,
			resources:  resources,
			applier:    app,
			remediator: rem,
		},
		discoveryInterface: dc,
		converter:          converter,
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

var _ Parser = &root{}

func (p *root) options() *opts {
	return &(p.opts)
}

// parseSource implements the Parser interface
func (p *root) parseSource(ctx context.Context, state gitState) ([]ast.FileObject, status.MultiError) {
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

	crds, err := p.declaredCRDs()
	if err != nil {
		return nil, err
	}
	builder := utildiscovery.ScoperBuilder(p.discoveryInterface)

	glog.Infof("Parsing files from git dir: %s", state.policyDir.OSPath())
	objs, err := p.parser.Parse(filePaths)
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

	if p.sourceFormat == filesystem.SourceFormatUnstructured {
		options.Visitors = append(options.Visitors, addImplicitNamespaces)
		objs, err = validate.Unstructured(objs, options)
	} else {
		objs, err = validate.Hierarchical(objs, options)
	}

	metrics.RecordReconcilerNonBlockingErrors(ctx, status.NonBlockingErrors(err))

	if status.HasBlockingErrors(err) {
		return nil, err
	}

	// Duplicated with namespace.go.
	e := addAnnotationsAndLabels(objs, declared.RootReconciler, p.gitContext(), state.commit)
	if e != nil {
		err = status.Append(err, status.InternalErrorf("unable to add annotations and labels: %v", e))
		return nil, err
	}
	return objs, err
}

// setSourceStatus implements the Parser interface
func (p *root) setSourceStatus(ctx context.Context, newStatus gitStatus) error {
	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		return status.APIServerError(err, "failed to get RootSync for parser")
	}

	cse := status.ToCSE(newStatus.errs)
	rs.Status.Source.Commit = newStatus.commit
	rs.Status.Source.Git = v1alpha1.GitStatus{
		Repo:     p.GitRepo,
		Revision: p.GitRev,
		Branch:   p.GitBranch,
		Dir:      p.PolicyDir.SlashPath(),
	}
	rs.Status.Source.Errors = cse
	rs.Status.Source.LastUpdate = newStatus.lastUpdate

	continueSyncing := true
	if len(cse) > 0 {
		continueSyncing = false
	}
	metrics.RecordPipelineError(ctx, RootSyncReconcilerType, "source", len(cse))
	rootsync.SetSyncing(&rs, continueSyncing, "Source", "Source", newStatus.commit, cse, newStatus.lastUpdate)

	metrics.RecordReconcilerErrors(ctx, "source", len(cse))

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RootSync source status from parser")
	}
	return nil
}

// setRenderingStatus implements the Parser interface
func (p *root) setRenderingStatus(ctx context.Context, oldStatus, newStatus renderingStatus) error {
	if oldStatus.equal(newStatus) {
		return nil
	}

	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		return status.APIServerError(err, "failed to get RootSync for parser")
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
	metrics.RecordPipelineError(ctx, RootSyncReconcilerType, "rendering", len(cse))

	rootsync.SetSyncing(&rs, continueSyncing, "Rendering", newStatus.message, newStatus.commit, cse, newStatus.lastUpdate)

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RootSync rendering status from parser")
	}
	return nil
}

// setSyncStatus implements the Parser interface
func (p *root) setSyncStatus(ctx context.Context, newStatus gitStatus) error {
	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		return status.APIServerError(err, "failed to get RootSync for parser")
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
	metrics.RecordPipelineError(ctx, RootSyncReconcilerType, "sync", len(syncErrs))
	metrics.RecordLastSync(ctx, newStatus.commit, newStatus.lastUpdate.Time)

	var allErrs []v1alpha1.ConfigSyncError
	allErrs = append(allErrs, rs.Status.Source.Errors...)
	allErrs = append(allErrs, syncErrs...)
	if len(allErrs) == 0 {
		rs.Status.LastSyncedCommit = newStatus.commit
	}
	rootsync.SetSyncing(&rs, false, "Sync", "Sync Completed", newStatus.commit, allErrs, newStatus.lastUpdate)

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RootSync sync status from parser")
	}
	return nil
}

// addImplicitNamespaces hydrates the given FileObjects by injecting implicit
// namespaces into the list before returning it. Implicit namespaces are those
// that are declared by an object's metadata namespace field but are not present
// in the list. Note that this function always returns a nil error to conform to
// the validate.VisitorFunc interface.
func addImplicitNamespaces(objs []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	// namespaces will track the set of Namespaces we expect to exist, and those
	// which actually do.
	namespaces := make(map[string]bool)

	for _, o := range objs {
		if o.GetObjectKind().GroupVersionKind().GroupKind() == kinds.Namespace().GroupKind() {
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
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(kinds.Namespace())
		u.SetName(ns)
		// We do NOT want to delete theses implicit Namespaces when the resources
		// inside them are removed from the repo. We don't know when it is safe to remove
		// the implicit namespaces. An implicit namespace may already exist in the
		// cluster. Deleting it will cause other unmanaged resources in that namespace
		// being deleted.
		//
		// Adding the LifecycleDeleteAnnotation is to prevent the applier from deleting
		// the implicit namespace when the namespaced config is removed from the repo.
		// Note that if the user later declares the
		// Namespace without this annotation, the annotation is removed as expected.
		u.SetAnnotations(map[string]string{common.LifecycleDeleteAnnotation: common.PreventDeletion})
		objs = append(objs, ast.NewFileObject(u, cmpath.RelativeOS("")))
	}

	return objs, nil
}
