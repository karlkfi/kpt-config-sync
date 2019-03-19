package repo

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"

	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
	syncclient "github.com/google/nomos/pkg/syncer/client"
	"k8s.io/apimachinery/pkg/runtime"
)

// Client wraps the syncer Client with functions specific to Repo objects.
type Client struct {
	client *syncclient.Client
	lister listersv1.RepoLister
}

// New returns a new Client.
func New(client *syncclient.Client, lister listersv1.RepoLister) *Client {
	return &Client{client: client, lister: lister}
}

// GetOrCreateRepo returns the Repo resource for the cluster or creates if it does not yet exist.
func (c *Client) GetOrCreateRepo(ctx context.Context) (*v1.Repo, status.Error) {
	repos, err := c.lister.List(labels.Everything())
	if err != nil {
		return nil, status.APIServerWrapf(err, "failed to list Repos")
	}
	// Repo is a singlteon so there should not be more than one.
	if len(repos) > 1 {
		resList := make([]id.Resource, len(repos))
		for i, r := range repos {
			resList[i] = ast.ParseFileObject(r)
		}
		return nil, id.MultipleSingletonsWrap(resList...)
	}
	if len(repos) == 1 {
		return repos[0].DeepCopy(), nil
	}

	repo, cErr := c.CreateRepo(ctx)
	if cErr != nil {
		return nil, cErr
	}
	return repo, nil // return explicit nil due to golang interfaces
}

// CreateRepo creates a new Repo resource for the cluster. Currently we don't do anything with the
// Repo object if a user has defined it in their source of truth so this is harmless/correct. If we
// start using it to drive logic then we may not want to be creating one here.
func (c *Client) CreateRepo(ctx context.Context) (*v1.Repo, id.ResourceError) {
	repoObj := Default()
	if err := c.client.Create(ctx, repoObj); err != nil {
		return nil, err
	}
	return repoObj, nil
}

// The following update functions are broken down by subsection of the overall RepoStatus to reduce
// chances of conflict/collision/overwrite.

// UpdateImportStatus updates the portion of the RepoStatus related to the importer.
func (c *Client) UpdateImportStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, id.ResourceError) {
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newRepo := obj.(*v1.Repo)
		newRepo.Status.Import = repo.Status.Import
		return newRepo, nil
	}
	newObj, err := c.client.UpdateStatus(ctx, repo, updateFn)
	if err != nil {
		return nil, err
	}
	return newObj.(*v1.Repo), nil
}

// UpdateSourceStatus updates the portion of the RepoStatus related to the source of truth.
func (c *Client) UpdateSourceStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, id.ResourceError) {
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newRepo := obj.(*v1.Repo)
		newRepo.Status.Source = repo.Status.Source
		return newRepo, nil
	}
	newObj, err := c.client.UpdateStatus(ctx, repo, updateFn)
	if err != nil {
		return nil, err
	}
	return newObj.(*v1.Repo), nil
}

// UpdateSyncStatus updates the portion of the RepoStatus related to the syncer.
func (c *Client) UpdateSyncStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, id.ResourceError) {
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newRepo := obj.(*v1.Repo)
		newRepo.Status.Sync = repo.Status.Sync
		return newRepo, nil
	}
	newObj, err := c.client.UpdateStatus(ctx, repo, updateFn)
	if err != nil {
		return nil, err
	}
	return newObj.(*v1.Repo), nil
}
