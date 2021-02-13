package ntopts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/pkg/client/restconfig"
)

// RemoteCluster tells the test to use the remote cluster pointed to by the config flags.
func RemoteCluster(t *testing.T) Opt {
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

		restConfig, err := restconfig.NewRestConfig()
		if err != nil {
			t.Fatal(err)
		}
		opt.RESTConfig = restConfig
	}
}
