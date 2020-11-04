package docker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"testing"
)

// RegistryName is the name of the local Docker registry.
const RegistryName = "kind-registry"

// RegistryPort is the port the local Docker registry is hosted on.
const RegistryPort = 5000

// StartLocalRegistry starts a local Docker registry if it is not running.
//
// To manually stop the repository (for whatever reason):
// $ docker stop kind-registry
//
// Assumes docker-registry.sh has already been run on the machine - otherwise
// calls t.Fatal.
func StartLocalRegistry(t *testing.T) {
	t.Helper()

	// Check if the registry is already running.
	out, err := exec.Command("docker", "inspect", "-f", "'{{.State.Running}}'", RegistryName).Output()
	if err != nil {
		t.Logf("docker inspect out: %q", string(out))
		t.Logf("docker inspect err: %v", err)
		t.Fatal("docker registry not configured or configured improperly; see e2e/doc.go")
	}
	switch strings.Trim(string(out), "\n'") {
	case "true":
		// The registry is already running, so nothing to do.
		return
	case "false":
		// The registry container exists but it isn't running, so start it.
		out, err := exec.Command("docker", "start", RegistryName).Output()
		if err != nil {
			t.Logf("docker start %s out: %q", RegistryName, out)
			t.Fatalf("docker start %s err: %v", RegistryName, err)
		}
		return
	default:
		// It isn't clear how this could be reached.
		t.Fatalf("unexpected docker inspect output: %q", string(out))
	}
}

// CheckImages ensures that all required images are installed on the local
// docker registry.
func CheckImages(t *testing.T) {
	t.Helper()

	var images = []string{
		"nomos",
		"reconciler",
		"reconciler-manager",
	}

	for _, image := range images {
		checkImage(t, image)
	}
}

func checkImage(t *testing.T, image string) {
	url := fmt.Sprintf("http://localhost:5000/%s:latest", image)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to check for image %s in regsitry: %s", image, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response for image %s in regsitry: %s", image, err)
	}
}
