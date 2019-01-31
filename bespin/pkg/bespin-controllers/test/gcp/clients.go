package gcp

import (
	"context"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
)

// UserAgent specifies the User Agent KCC presents to GCP.
const UserAgent = "kcc/controller-manager"

// NewCloudResourceManagerClient returns a GCP Cloud Resource Manager service.
func NewCloudResourceManagerClient(ctx context.Context) (*cloudresourcemanager.Service, error) {
	httpClient, err := google.DefaultClient(ctx, cloudresourcemanager.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	client, err := cloudresourcemanager.New(httpClient)
	if err != nil {
		return nil, err
	}
	client.UserAgent = UserAgent
	return client, nil
}
