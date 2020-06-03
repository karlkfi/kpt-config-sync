package nomostest

import (
	"fmt"
	"os/exec"
	"strings"

	v1 "k8s.io/api/core/v1"
)

const registryName = "kind-registry"
const registryPort = 5000

// startDockerRegistry starts a local Docker registry if it does not exist.
//
// Adapted from https://kind.sigs.k8s.io/docs/user/local-registry/
// Roughly equivalent to:
// running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
//if [ "${running}" != 'true' ]; then
//  docker run \
//    -d --restart=always -p "${reg_port}:5000" --name "${reg_name}" \
//    registry:2
//fi
//
// To manually stop the repository (for whatever reason):
// $ docker stop kind-registry
func startLocalRegistry(nt *NT) {
	// Check if the registry is already running.
	out, err := exec.Command("docker", "inspect", "-f", "'{{.State.Running}}'", registryName).Output()
	if err == nil {
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
	nt.T.Logf("docker inspect out: %q", string(out))
	nt.T.Logf("docker inspect err: %v", err)

	// The registry is not active.
	out, err = exec.Command("docker", "run", "-d", "--restart", "always",
		"-p", fmt.Sprintf("%d:5000", registryPort), "--name", registryName, "registry:2",
	).CombinedOutput()
	if err != nil {
		nt.T.Logf("docker run out: %q", string(out))
		nt.T.Fatalf("docker run err: %v", err)
	}
	// TODO(willbeason): Automatically shut this down after go test completes.
	//  Note: This must be done by the caller; it can't be done here.
}

func connectToLocalRegistry(nt *NT) {
	nt.T.Helper()
	// TODO(willbeason): Push the ConfigSync images to the image repository.
	startLocalRegistry(nt)

	// See https://kind.sigs.k8s.io/docs/user/local-registry/ for explanation.
	node := &v1.Node{}
	err := nt.Get(nt.Name+"-control-plane", "", node)
	if err != nil {
		nt.T.Fatal(err)
	}
	node.Annotations["kind.x-k8s.io/registry"] = fmt.Sprintf("localhost:%d", registryPort)
	err = nt.Update(node)
	if err != nil {
		nt.T.Fatal(err)
	}
}
