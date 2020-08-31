package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
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
	gitDir cmpath.Absolute,
	policyDir cmpath.Relative,
	gitRef string,
	gitRepo string,
	discoveryInterfaceGetter discovery.ClientGetter,
) (Runnable, error) {
	opts := opts{
		clusterName:      clusterName,
		client:           c,
		pollingFrequency: pollingFrequency,
		files: files{
			gitDir:    gitDir,
			policyDir: policyDir,
			gitRef:    gitRef,
			gitRepo:   gitRepo,
		},
	}

	switch format {
	case filesystem.SourceFormatUnstructured:
		opts.parser = filesystem.NewParser(fileReader, discoveryInterfaceGetter)
	case filesystem.SourceFormatHierarchy:
		opts.parser = filesystem.NewRawParser(fileReader, discoveryInterfaceGetter)
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
			err := p.Parse(ctx)
			if err != nil {
				glog.Error(err)
			}
			p.setRootSyncErrs(ctx, err)
		}
	}
}

// Parse implements Runnable.
func (p *root) Parse(ctx context.Context) status.MultiError {
	policyDir, wantFiles, err := p.absPolicyDir()
	if err != nil {
		return err
	}
	if p.lastApplied == policyDir.OSPath() {
		return nil
	}

	if p.sourceFormat == filesystem.SourceFormatHierarchy {
		// We're using hierarchical mode for the root repository, so ignore files
		// outside of the allowed directories.
		wantFiles = filesystem.FilterHierarchyFiles(policyDir, wantFiles)
	}

	glog.Infof("Parsing files from git dir: %s", policyDir.OSPath())
	cos, err := p.parser.Parse(p.clusterName, true, listCrds(ctx, p.client), policyDir, wantFiles)
	if err != nil {
		return err
	}

	commitHash, e := p.CommitHash()
	if e != nil {
		err = status.Append(err, e)
		return err
	}

	addAnnotationsAndLabels(cos, declared.RootReconciler, p.gitRef, p.gitRepo, commitHash)

	// TODO(b/163053203): Validate RepoSync CRs.
	err = p.update(ctx, cos)
	if err == nil {
		glog.V(4).Infof("Successfully applied all files from git dir: %s", policyDir.OSPath())
		p.lastApplied = policyDir.OSPath()
	}
	return err
}

func (p *root) setRootSyncErrs(ctx context.Context, errs status.MultiError) {
	var rs v1.RootSync
	if err := p.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		glog.Errorf("Failed to get RootSync for parser: %v", err)
		return
	}

	rs.Status.Sync.LastUpdate = metav1.Now()
	rs.Status.Sync.Errors = status.ToCSE(errs)
	if err := p.client.Status().Update(ctx, &rs); err != nil {
		glog.Errorf("Failed to update RootSync status from parser: %v", err)
	}
}
