package repo

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	syncclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		return nil, status.MultipleSingletonsWrap(resList...)
	}
	if len(repoList.Items) == 1 {
		return setTypeMeta(repoList.Items[0].DeepCopy()), nil
	}

	return c.CreateRepo(ctx)
}

// CreateRepo creates a new Repo resource for the cluster. Currently we don't do anything with the
// Repo object if a user has defined it in their source of truth so this is harmless/correct. If we
// start using it to drive logic then we may not want to be creating one here.
func (c *Client) CreateRepo(ctx context.Context) (*v1.Repo, status.Error) {
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
func (c *Client) UpdateImportStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, status.Error) {
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
func (c *Client) UpdateSourceStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, status.Error) {
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
func (c *Client) UpdateSyncStatus(ctx context.Context, repo *v1.Repo) (*v1.Repo, status.Error) {
	if c.importerClient != nil {
		return c.importerClient.updateRepo(repo)
	}
	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newRepo := obj.(*v1.Repo)
		if cmp.Equal(repo.Status.Sync, newRepo.Status.Sync) {
			return nil, action.NoUpdateNeeded()
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

// genClient is an internal client that uses generated informer classes to perform CRUD operations
// on Repos. It is strictly here to support the importer until that process migrates from informers
// to manager/controllers.
type genClient struct {
	client typedv1.RepoInterface
	lister listersv1.RepoLister
}

func (c *genClient) getRepo() (*v1.Repo, status.Error) {
	repos, err := c.lister.List(labels.Everything())
	if err != nil {
		return nil, status.APIServerWrapf(err, "failed to list Repos")
	}
	// Repo is a singleton so there should not be more than one.
	if len(repos) > 1 {
		resList := make([]id.Resource, len(repos))
		for i, r := range repos {
			resList[i] = ast.ParseFileObject(r)
		}
		return nil, status.MultipleSingletonsWrap(resList...)
	}
	if len(repos) == 1 {
		return setTypeMeta(repos[0].DeepCopy()), nil
	}
	return nil, nil
}

func (c *genClient) getOrCreateRepo() (*v1.Repo, status.Error) {
	repo, err := c.getRepo()
	if err != nil {
		return nil, err
	}
	if repo != nil {
		return repo, nil
	}

	return c.createRepo()
}

func (c *genClient) createRepo() (*v1.Repo, status.Error) {
	repoObj := Default()
	createdObj, err := c.client.Create(repoObj)
	if err != nil {
		return nil, status.ResourceWrap(err, "failed to create Repo", ast.ParseFileObject(repoObj))
	}
	return setTypeMeta(createdObj), nil
}

func (c *genClient) updateRepo(repoObj *v1.Repo) (*v1.Repo, status.Error) {
	var lastError status.Error
	retryBackoff := 1 * time.Millisecond
	maxTries := 5

	for tryNum := 0; tryNum < maxTries; tryNum++ {
		existingRepo, sErr := c.getRepo()
		if sErr != nil {
			return nil, status.MissingResourceWrap(sErr, "failed to get repo to update", ast.ParseFileObject(repoObj))
		}
		if existingRepo == nil {
			return nil, status.MissingResourceWrap(errors.New("failed to get repo to update"), "", ast.ParseFileObject(repoObj))
		}

		existingRepo.Status.Source = repoObj.Status.Source
		existingRepo.Status.Import = repoObj.Status.Import
		newObj, err := c.client.UpdateStatus(existingRepo)
		if err == nil {
			return setTypeMeta(newObj), nil
		}

		lastError = status.ResourceWrap(err, "failed to update repo", ast.ParseFileObject(existingRepo))
		if !apierrors.IsConflict(err) {
			return nil, lastError
		}
		glog.Infof("Conflict on update: %v", err)
		<-time.After(retryBackoff)
		retryBackoff += 1 * time.Millisecond
	}
	return nil, lastError
}
