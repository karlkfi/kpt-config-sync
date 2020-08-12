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

	cos, err := p.parser.Parse(p.clusterName, true, listCrds(ctx, p.reader), policyDir, wantFiles)
	if err != nil {
		return err
	}
	objs := filesystem.AsFileObjects(cos)

	scoper, err := p.buildScoper(ctx)
	if err != nil {
		return err
	}
	// We recreate this validator with every run as the set of available CRDs may
	// change between runs. The user may have either declared new CRDs in the root
	// repo, or they may have manually applied new ones.
	err = noClusterScopeValidator(scoper).Validate(objs)
	if err != nil {
		return err
	}

	nsv := repositoryScopeVisitor(p.scope)
	err = nsv.Validate(objs)
	if err != nil {
		return err
	}

	return p.update(ctx, cos)
}

func (p *namespace) buildScoper(ctx context.Context) (discovery.Scoper, status.MultiError) {
	// Initialize the scoper with the core Kubernetes types.
	scoper := discovery.CoreScoper()
	// Add any CRDs currently available on the cluster.
	//
	// There is a race condition here, as we can't guarantee the Root parser has
	// fully synced all declared CRDs. Recall that the namespace parsers are
	// running asynchronously.
	crds, err := listCrds(ctx, p.reader)()
	if err != nil {
		return nil, err
	}
	scoper.AddCustomResources(crds)
	return scoper, nil
}
