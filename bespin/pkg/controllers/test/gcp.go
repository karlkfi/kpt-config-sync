package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/nomos/bespin/pkg/controllers/test/gcp"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
)

// GetDefaultProjectID returns the project set by gcloud.
func GetDefaultProjectID(t *testing.T) string {
	t.Helper()
	projectID, err := gcp.GetDefaultProjectID()
	if err != nil {
		t.Fatalf("error retrieving gcloud sdk credentials: %v", err)
	}
	return projectID
}

// GetDefaultServiceAccount returns the GCP default credentials.
func GetDefaultServiceAccount(t *testing.T) string {
	creds, err := google.FindDefaultCredentials(context.TODO(), cloudresourcemanager.CloudPlatformScope)
	if err != nil {
		t.Fatalf("error getting credentials: %v", err)
	}
	if creds == nil {
		t.Fatalf("test running in environment without default credentials, can't continue")
	}

	var rawCreds map[string]string
	if err := json.Unmarshal(creds.JSON, &rawCreds); err != nil {
		t.Fatalf("creds file malformed: %v", err)
	}

	return rawCreds["client_email"]
}
