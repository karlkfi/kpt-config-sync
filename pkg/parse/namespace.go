package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewNamespaceParser creates a new runnable parser for parsing a Namespace repo.
func NewNamespaceParser(
	scope declared.Scope,
	fileReader filesystem.Reader,
	c client.Client,
	pollingFrequency time.Duration,
	gitDir cmpath.Absolute,
	policyDir cmpath.Relative,
	gitRef string,
	gitRepo string,
	discoveryInterfaceGetter discovery.ClientGetter,
) Runnable {
	return &namespace{
		opts: opts{
			client:           c,
			pollingFrequency: pollingFrequency,
			files: files{
				gitDir:    gitDir,
				policyDir: policyDir,
				gitRef:    gitRef,
				gitRepo:   gitRepo,
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
	scope declared.Scope
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
			p.setRepoSyncErrs(ctx, err)
		}
	}
}

// Parse implements Runnable.
func (p *namespace) Parse(ctx context.Context) status.MultiError {
	policyDir, wantFiles, err := p.absPolicyDir()
	if err != nil {
		return err
	}

	cos, err := p.parser.Parse(p.clusterName, true, listCrds(ctx, p.client), policyDir, wantFiles)
	if err != nil {
		return err
	}

	// Parse and generate a ResourceGroup from the Kptfile if it exists
	cos, e := AsResourceGroup(cos)
	if e != nil {
		err = status.Append(err, e)
		return err
	}

	commitHash, e := p.CommitHash()
	if e != nil {
		err = status.Append(err, e)
		return err
	}
	addAnnotationsAndLabels(cos, p.scope, p.gitRef, p.gitRepo, commitHash)

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
	crds, err := listCrds(ctx, p.client)()
	if err != nil {
		return nil, err
	}
	scoper.AddCustomResources(crds)
	return scoper, nil
}

func (p *namespace) setRepoSyncErrs(ctx context.Context, errs status.MultiError) {
	var rs v1.RepoSync
	if err := p.client.Get(ctx, reposync.ObjectKey(p.scope), &rs); err != nil {
		glog.Errorf("Failed to get RepoSync for %s parser: %v", p.scope, err)
		return
	}

	rs.Status.Sync.LastUpdate = metav1.Now()
	rs.Status.Sync.Errors = status.ToCSE(errs)
	if err := p.client.Status().Update(ctx, &rs); err != nil {
		glog.Errorf("Failed to update RepoSync status from %s parser: %v", p.scope, err)
	}
}
