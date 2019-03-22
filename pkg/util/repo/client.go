package repo

import (
	"context"

	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
	syncclient "github.com/google/nomos/pkg/syncer/client"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps the syncer Client with functions specific to Repo objects.
type Client struct {
	client *syncclient.Client
	// TODO: remove this client once the importer has migrated from informers to manager/controllers
	importerClient *genClient
}

// New returns a new Client.
func New(client *syncclient.Client) *Client {
	return &Client{client: client}
}

// NewForImporter returns a new Client specifically for the importer process.
func NewForImporter(client typedv1.RepoInterface, lister listersv1.RepoLister) *Client {
	importerClient := &genClient{client: client, lister: lister}
	return &Client{importerClient: importerClient}
}

// GetOrCreateRepo returns the Repo resource for the cluster or creates if it does not yet exist.
func (c *Client) GetOrCreateRepo(ctx context.Context) (*v1.Repo, status.Error) {
	if c.importerClient != nil {
		return c.importerClient.getOrCreateRepo()
	}
	var repoList v1.RepoList
	if err := c.client.List(ctx, &client.ListOptions{}, &repoList); err != nil {
		return nil, status.APIServerWrapf(err, "failed to list Repos")
	}
	if len(repoList.Items) > 1 {
		resList := make([]id.Resource, len(repoList.Items))
		for i, r := range repoList.Items {
			resList[i] = ast.ParseFileObject(&r)
		}
		return nil, id.MultipleSingletonsWrap(resList...)
	}
	if len(repoList.Items) == 1 {
		return repoList.Items[0].DeepCopy(), nil
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
	if c.importerClient != nil {
		return c.importerClient.createRepo()
	}
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
	if c.importerClient != nil {
		return c.importerClient.updateRepo(repo)
	}
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
	if c.importerClient != nil {
		return c.importerClient.updateRepo(repo)
	}
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
	if c.importerClient != nil {
		return c.importerClient.updateRepo(repo)
	}
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

// genClient is an internal client that uses generated informer classes to perform CRUD operations
// on Repos. It is strictly here to support the importer until that process migrates from informers
// to manager/controllers.
type genClient struct {
	client typedv1.RepoInterface
	lister listersv1.RepoLister
}

func (c *genClient) getOrCreateRepo() (*v1.Repo, status.Error) {
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

	repo, cErr := c.createRepo()
	if cErr != nil {
		return nil, cErr
	}
	return repo, nil // return explicit nil due to golang interfaces
}

func (c *genClient) createRepo() (*v1.Repo, id.ResourceError) {
	repoObj := Default()
	createdObj, err := c.client.Create(repoObj)
	if err != nil {
		return nil, id.ResourceWrap(err, "failed to create Repo", ast.ParseFileObject(repoObj))
	}
	return createdObj, nil
}

func (c *genClient) updateRepo(repoObj *v1.Repo) (*v1.Repo, id.ResourceError) {
	newObj, err := c.client.Update(repoObj)
	if err != nil {
		return nil, id.ResourceWrap(err, "failed to update Repo", ast.ParseFileObject(repoObj))
	}
	return newObj, nil
}
