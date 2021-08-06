package gitproviders

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/google/nomos/e2e"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

const (
	bitbucketProject = "CSCI"

	// PrivateSSHKey is secret name of the private SSH key stored in the Cloud Secret Manager.
	PrivateSSHKey = "config-sync-ci-ssh-private-key"

	// secretManagerProject is the project id of the Secret Manager that stores the secrets.
	secretManagerProject = "stolos-dev"
)

// BitbucketClient is the client that calls the Bitbucket REST APIs.
type BitbucketClient struct {
	oauthKey     string
	oauthSecret  string
	refreshToken string
}

// newBitbucketClient instantiates a new Bitbucket client.
func newBitbucketClient() (*BitbucketClient, error) {
	client := &BitbucketClient{}

	var err error
	if client.oauthKey, err = FetchCloudSecret("bitbucket-oauth-key"); err != nil {
		return client, err
	}
	if client.oauthSecret, err = FetchCloudSecret("bitbucket-oauth-secret"); err != nil {
		return client, err
	}
	if client.refreshToken, err = FetchCloudSecret("bitbucket-refresh-token"); err != nil {
		return client, err
	}
	return client, nil
}

// Type returns the provider type.
func (b *BitbucketClient) Type() string {
	return e2e.Bitbucket
}

// RemoteURL returns the Git URL for the Bitbucket repository.
func (b *BitbucketClient) RemoteURL(_ int, repoName string) string {
	return b.SyncURL(repoName)
}

// SyncURL returns a URL for Config Sync to sync from.
func (b *BitbucketClient) SyncURL(repoName string) string {
	return fmt.Sprintf("git@bitbucket.org:%s/%s", GitUser, repoName)
}

// CreateRepository calls the POST API to create a remote repository on Bitbucket.
// The remote repo name is unique with a prefix of the local name.
func (b *BitbucketClient) CreateRepository(localName string) (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate a new UUID")
	}
	// Make the remote repoName unique in order to run multiple tests in parallel.
	repoName := localName + "-" + u.String()

	// Create a remote repository on demand with a random localName.
	accessToken, err := b.refreshAccessToken()
	if err != nil {
		return "", err
	}

	out, err := exec.Command("curl", "-sX", "POST",
		"-H", "Content-Type: application/json",
		"-H", fmt.Sprintf("Authorization:Bearer %s", accessToken),
		"-d", fmt.Sprintf("{\"scm\": \"git\",\"project\": {\"key\": \"%s\"},\"is_private\": \"true\"}", bitbucketProject),
		fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s", GitUser, repoName)).CombinedOutput()

	if err != nil {
		return "", errors.Wrap(err, string(out))
	}
	return repoName, nil
}

// DeleteRepositories calls the DELETE API to delete all remote repositories on Bitbucket.
// It deletes multiple repos in a single function in order to reuse the access_token.
func (b *BitbucketClient) DeleteRepositories(names ...string) error {
	accessToken, err := b.refreshAccessToken()
	if err != nil {
		return err
	}

	var errs error
	for _, name := range names {
		out, err := exec.Command("curl", "-sX", "DELETE",
			"-H", fmt.Sprintf("Authorization:Bearer %s", accessToken),
			fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s",
				GitUser, name)).CombinedOutput()

		if err != nil {
			errs = multierr.Append(errs, errors.Wrap(err, string(out)))
		}
		if len(out) != 0 {
			errs = multierr.Append(errs, errors.New(string(out)))
		}
	}
	return errs
}

func (b *BitbucketClient) refreshAccessToken() (string, error) {
	out, err := exec.Command("curl", "-sX", "POST", "-u",
		fmt.Sprintf("%s:%s", b.oauthKey, b.oauthSecret),
		"https://bitbucket.org/site/oauth2/access_token",
		"-d", "grant_type=refresh_token",
		"-d", "refresh_token="+b.refreshToken).CombinedOutput()

	if err != nil {
		return "", errors.Wrap(err, string(out))
	}

	var output map[string]interface{}
	err = json.Unmarshal(out, &output)
	if err != nil {
		return "", err
	}

	accessToken, ok := output["access_token"]
	if !ok {
		return "", fmt.Errorf("no access_token: %s", string(out))
	}

	return accessToken.(string), nil
}

// FetchCloudSecret fetches secret from Google Cloud Secret Manager.
func FetchCloudSecret(name string) (string, error) {
	out, err := exec.Command("gcloud", "secrets", "versions",
		"access", "latest", "--project", secretManagerProject, "--secret", name).CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, string(out))
	}
	return string(out), nil
}
