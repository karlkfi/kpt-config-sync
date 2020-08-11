package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRootParser creates a new runnable parser for parsing a Root repository.
func NewRootParser(
	clusterName string,
	format filesystem.SourceFormat,
	fileReader filesystem.Reader,
	reader client.Reader,
	pollingFrequency time.Duration,
	gitDir cmpath.Absolute,
	policyDir cmpath.Relative,
	discoveryInterfaceGetter discovery.ClientGetter,
) (Runnable, error) {
	opts := opts{
		clusterName:      clusterName,
		reader:           reader,
		pollingFrequency: pollingFrequency,
		files: files{
			gitDir:    gitDir,
			policyDir: policyDir,
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
		}
	}
}

// Parse implements Runnable.
func (p *root) Parse(ctx context.Context) status.MultiError {
	policyDir, wantFiles, err := p.absPolicyDir()
	if err != nil {
		return err
	}
	if p.sourceFormat == filesystem.SourceFormatHierarchy {
		// We're using hierarchical mode for the root repository, so ignore files
		// outside of the allowed directories.
		wantFiles = filesystem.FilterHierarchyFiles(policyDir, wantFiles)
	}

	objs, err := p.parser.Parse(p.clusterName, true, listCrds(ctx, p.reader), policyDir, wantFiles)
	if err != nil {
		return err
	}

	// TODO(b/163053203): Validate RepoSync CRs.
	return p.update(ctx, objs)
}
