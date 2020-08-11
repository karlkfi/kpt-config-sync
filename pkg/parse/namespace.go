package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewNamespaceParser creates a new runnable parser for parsing a Namespace repo.
func NewNamespaceParser(
	scope string,
	fileReader filesystem.Reader,
	clientReader client.Reader,
	pollingFrequency time.Duration,
	gitDir cmpath.Absolute,
	policyDir cmpath.Relative,
	discoveryInterfaceGetter discovery.ClientGetter,
) Runnable {
	return &namespace{
		opts: opts{
			reader:           clientReader,
			pollingFrequency: pollingFrequency,
			files: files{
				gitDir:    gitDir,
				policyDir: policyDir,
			},
			parser: filesystem.NewRawParser(fileReader, discoveryInterfaceGetter),
		},
		scope: scope,
	}
}

type namespace struct {
	opts

	// scope is the name of the Namespace this parser is for.
	// It is an error for this parser's repository to contain resources outside of
	// this Namespace.
	scope string
}

// Run implements Runnable.
func (p *namespace) Run(ctx context.Context) {
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
func (p *namespace) Parse(ctx context.Context) status.MultiError {
	policyDir, wantFiles, err := p.absPolicyDir()
	if err != nil {
		return err
	}

	objs, err := p.parser.Parse(p.clusterName, true, listCrds(ctx, p.reader), policyDir, wantFiles)
	if err != nil {
		return err
	}

	nsv := namespaceScopeVisitor(p.scope)
	err = nsv.Validate(objs)
	if err != nil {
		return err
	}

	return p.update(ctx, objs)
}
