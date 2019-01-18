package gcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
)

// GetDefaultProjectID retrieves the current project specified by the
// default GCP credentials on the host machine.
func GetDefaultProjectID() (string, error) {
	creds, err := google.FindDefaultCredentials(context.Background(), cloudresourcemanager.CloudPlatformScope)
	if err != nil {
		return "", fmt.Errorf("error retrieving gcp sdk credentials: %v", err)
	}
	if creds.ProjectID != "" {
		return creds.ProjectID, nil
	}
	return getGCloudDefaultProjectID()
}

func getGCloudDefaultProjectID() (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	bytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error executing command '%v': %v'", cmd, err)
	}
	return strings.TrimSpace(string(bytes)), nil
}
