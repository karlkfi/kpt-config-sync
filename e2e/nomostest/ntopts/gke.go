package ntopts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/pkg/client/restconfig"
)

// GKECluster tells the test to use the GKE cluster pointed to by the config flags.
func GKECluster(t *testing.T) Opt {
	return func(opt *New) {
		t.Helper()

		kubeconfig := *e2e.KubeConfig
		if len(kubeconfig) == 0 {
			home, err := os.UserHomeDir()
			if err != nil {
				t.Fatal(err)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		if err := os.Setenv(Kubeconfig, kubeconfig); err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		forceAuthRefresh(t)
		restConfig, err := restconfig.NewRestConfig()
		if err != nil {
			t.Fatal(err)
		}
		opt.RESTConfig = restConfig
	}
}

// forceAuthRefresh forces gcloud to refresh the access_token to avoid using an expired one in the middle of a test.
func forceAuthRefresh(t *testing.T) {
	out, err := exec.Command("kubectl", "config", "current-context").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get current config: %v", err)
	}
	context := strings.TrimSpace(string(out))
	gkeArgs := strings.Split(context, "_")
	if len(gkeArgs) < 4 {
		t.Fatalf("unknown GKE context fomrat: %s", context)
	}
	gkeProject := gkeArgs[1]
	gkeZone := gkeArgs[2]
	gkeCluster := gkeArgs[3]

	_, err = exec.Command("gcloud", "container", "clusters", "get-credentials",
		gkeCluster, "--zone", gkeZone, "--project", gkeProject).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get credentials: %v", err)
	}

	_, err = exec.Command("gcloud", "config", "config-helper", "--force-auth-refresh").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to refresh access_token: %v", err)
	}
}
