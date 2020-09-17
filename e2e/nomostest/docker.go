package nomostest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
)

const registryName = "kind-registry"
const registryPort = 5000

// startDockerRegistry starts a local Docker registry if it is not running.
//
// To manually stop the repository (for whatever reason):
// $ docker stop kind-registry
//
// Assumes docker-registry.sh has already been run on the machine - otherwise
// calls t.Fatal.
func startLocalRegistry(nt *NT) {
	nt.T.Helper()

	// Check if the registry is already running.
	out, err := exec.Command("docker", "inspect", "-f", "'{{.State.Running}}'", registryName).Output()
	if err != nil {
		nt.T.Logf("docker inspect out: %q", string(out))
		nt.T.Logf("docker inspect err: %v", err)
		nt.T.Fatal("docker registry not configured or configured improperly; see e2e/doc.go")
	}
	switch strings.Trim(string(out), "\n'") {
	case "true":
		// The registry is already running, so nothing to do.
		return
	case "false":
		// The registry container exists but it isn't running, so start it.
		out, err := exec.Command("docker", "start", registryName).Output()
		if err != nil {
			nt.T.Logf("docker start %s out: %q", registryName, out)
			nt.T.Fatalf("docker start %s err: %v", registryName, err)
		}
		return
	default:
		// It isn't clear how this could be reached.
		nt.T.Fatalf("unexpected docker inspect output: %q", string(out))
	}
}

func connectToLocalRegistry(nt *NT) {
	nt.T.Helper()
	startLocalRegistry(nt)

	// We get access to the kubectl API before the Kind cluster is finished being
	// set up, so the control plane is sometimes still being modified when we do
	// this.
	_, err := Retry(20*time.Second, func() error {
		// See https://kind.sigs.k8s.io/docs/user/local-registry/ for explanation.
		node := &v1.Node{}
		err := nt.Get(nt.ClusterName+"-control-plane", "", node)
		if err != nil {
			return err
		}
		node.Annotations["kind.x-k8s.io/registry"] = fmt.Sprintf("localhost:%d", registryPort)
		return nt.Update(node)
	})
	if err != nil {
		nt.T.Fatalf("connecting cluster to local Docker registry: %v", err)
	}
}

func checkDockerImages(nt *NT) {
	nt.T.Helper()

	var images = []string{
		"nomos",
		"reconciler",
		"reconciler-manager",
	}

	for _, image := range images {
		checkImage(nt, image)
	}
}

func checkImage(nt *NT, image string) {
	url := fmt.Sprintf("http://localhost:5000/%s:latest", image)
	resp, err := http.Get(url)
	if err != nil {
		nt.T.Fatalf("Failed to check for image %s in regsitry: %s", image, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		nt.T.Fatalf("Failed to read response for image %s in regsitry: %s", image, err)
	}
}
