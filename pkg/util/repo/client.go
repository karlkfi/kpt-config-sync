package repo

import (
	"context"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	syncclient "github.com/google/nomos/pkg/syncer/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps the syncer Client with functions specific to Repo objects.
type Client struct {
	client *syncclient.Client
}

// New returns a new Client.
func New(client *syncclient.Client) *Client {
	return &Client{client: client}
}

// GetOrCreateRepo returns the Repo resource for the cluster or creates if it does not yet exist.
func (c *Client) GetOrCreateRepo(ctx context.Context) (*v1.Repo, status.Error) {
	var repoList v1.RepoList
	if err := c.client.List(ctx, &repoList); err != nil {
		return nil, status.APIServerError(err, "failed to list Repos")
	}
	if len(repoList.Items) > 1 {
		resList := make([]client.Object, len(repoList.Items))
		for i, r := range repoList.Items {
			resList[i] = &r
		}
		return nil, status.MultipleSingletonsError(resList...)
	}
	if len(repoList.Items) == 1 {
		return setTypeMeta(repoList.Items[0].DeepCopy()), nil
	}

	return c.createRepo(ctx)
}

// createRepo creates a new Repo resource for the cluster. Currently we don't do anything with the
// Repo object if a user has defined it in their source of truth so this is harmless/correct. If we
// start using it to drive logic then we may not want to be creating one here.
func (c *Client) createRepo(ctx context.Context) (*v1.Repo, status.Error) {
	repoObj := Default()
	if err := c.client.Create(ctx, repoObj); err != nil {
		return nil, err
	}
	return repoObj, nil
}

// The following update functions are broken down by subsection of the overall RepoStatus to reduce
// chances of conflict/collision/overwrite.

// UpdateImportStatus updates the portion of the RepoStatus related to the importer.
func (c *Client) UpdateImportStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, status.Error) {
	updateFn := func(obj client.Object) (client.Object, error) {
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
func (c *Client) UpdateSourceStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, status.Error) {
	updateFn := func(obj client.Object) (client.Object, error) {
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
func (c *Client) UpdateSyncStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, status.Error) {
	updateFn := func(obj client.Object) (client.Object, error) {
		newRepo := obj.(*v1.Repo)
		if cmp.Equal(repo.Status.Sync, newRepo.Status.Sync) {
			return newRepo, syncclient.NoUpdateNeeded()
		}
		newRepo.Status.Sync = repo.Status.Sync
		return newRepo, nil
	}
	newObj, err := c.client.UpdateStatus(ctx, repo, updateFn)
	if err != nil {
		return nil, err
	}
	return newObj.(*v1.Repo), nil
}
