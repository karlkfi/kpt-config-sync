package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRootRunner creates a new runnable parser for parsing a Root repository.
func NewRootRunner(clusterName, reconcilerName string, format filesystem.SourceFormat, fileReader reader.Reader, c client.Client, pollingFrequency time.Duration, resyncPeriod time.Duration, fs FileSource, dc discovery.ServerResourcer, resources *declared.Resources, app kptapplier.Interface, rem remediator.Interface) (Parser, error) {
	opts := opts{
		clusterName:      clusterName,
		reconcilerName:   reconcilerName,
		client:           c,
		pollingFrequency: pollingFrequency,
		resyncPeriod:     resyncPeriod,
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
		opts.parser = filesystem.NewRawParser(fileReader, true, metav1.NamespaceDefault, declared.RootReconciler)
	case filesystem.SourceFormatHierarchy:
		opts.parser = filesystem.NewParser(fileReader, true)
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
	builder := discovery.ScoperBuilder(p.discoveryInterface)

	glog.Infof("Parsing files from git dir: %s", state.policyDir.OSPath())
	start := time.Now()
	objs, err := p.parser.Parse(p.clusterName, crds, builder, filePaths)
	metrics.RecordParseErrorAndDuration(ctx, err, start)
	if err != nil {
		return nil, err
	}

	// TODO(b/167700852): Inject this function into the Parser.
	if p.sourceFormat == filesystem.SourceFormatUnstructured {
		objs = addImplicitNamespaces(objs)
	}

	// Duplicated with namespace.go.
	e := addAnnotationsAndLabels(objs, declared.RootReconciler, p.gitContext(), state.commit)
	if e != nil {
		err = status.Append(err, status.InternalErrorf("unable to add annotations and labels: %v", e))
		return nil, err
	}
	return objs, nil
}

// setSourceStatus implements the Parser interface
func (p *root) setSourceStatus(ctx context.Context, oldStatus, newStatus gitStatus) error {
	if oldStatus.equal(newStatus) {
		return nil
	}

	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		return status.APIServerError(err, "failed to get RootSync for parser")
	}

	cse := status.ToCSE(newStatus.errs)
	rs.Status.Source.Commit = newStatus.commit
	rs.Status.Source.Errors = cse

	metrics.RecordReconcilerErrors(ctx, "source", len(cse))

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RootSync source status from parser")
	}
	return nil
}

// setSyncStatus implements the Parser interface
func (p *root) setSyncStatus(ctx context.Context, oldStatus, newStatus gitStatus) error {
	if oldStatus.equal(newStatus) {
		return nil
	}

	var rs v1alpha1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		return status.APIServerError(err, "failed to get RootSync for parser")
	}

	now := metav1.Now()
	cse := status.ToCSE(newStatus.errs)
	rs.Status.Sync.Commit = newStatus.commit
	rs.Status.Sync.Errors = cse
	rs.Status.Sync.LastUpdate = now

	metrics.RecordReconcilerErrors(ctx, "sync", len(cse))
	metrics.RecordLastSync(ctx, now.Time)

	if err := p.client.Status().Update(ctx, &rs); err != nil {
		return status.APIServerError(err, "failed to update RootSync sync status from parser")
	}
	return nil
}

func addImplicitNamespaces(objs []ast.FileObject) []ast.FileObject {
	// namespaces will track the set of Namespaces we expect to exist, and those
	// which actually do.
	namespaces := make(map[string]bool)

	for _, o := range objs {
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
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(kinds.Namespace())
		u.SetName(ns)
		// We do NOT want to delete theses implicit Namespaces when the resources
		// inside them are removed. Note that if the user later declares the
		// Namespace without this annotation, the annotation is removed as expected.
		u.SetAnnotations(map[string]string{common.LifecycleDeleteAnnotation: common.PreventDeletion})
		objs = append(objs, ast.NewFileObject(u, cmpath.RelativeOS("")))
	}

	return objs
}
