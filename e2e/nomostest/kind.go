package nomostest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

const (
	// kubeconfig is the filename of the KUBECONFIG file.
	kubeconfig = "KUBECONFIG"

	// maxKindTries is the number of times to attempt to create a Kind cluster for
	// a single test.
	maxKindTries = 6
)

func createKindCluster(p *cluster.Provider, name, kcfgPath string, version ntopts.KindVersion) error {
	var err error
	for i := 0; i < maxKindTries; i++ {
		if i > 0 {
			// This isn't the first time we're executing this loop.
			// We've tried creating the cluster before but got an error. Since we set
			// retain=true, the cluster still exists in a problematic state. We must
			// delete is before retrying.
			err = p.Delete(name, kcfgPath)
			if err != nil {
				return err
			}
		}

		err = p.Create(name,
			// Use Kubernetes 1.14
			// TODO(willbeason): Allow specifying Kubernetes version.
			cluster.CreateWithNodeImage(string(version)),
			// Store the KUBECONFIG at the specified path.
			cluster.CreateWithKubeconfigPath(kcfgPath),
			// Allow the cluster to see the local Docker container registry.
			// https://kind.sigs.k8s.io/docs/user/local-registry/
			cluster.CreateWithV1Alpha4Config(&v1alpha4.Cluster{
				ContainerdConfigPatches: []string{
					fmt.Sprintf(`[plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:%d"]
  endpoint = ["http://%s:%d"]`, registryPort, registryName, registryPort),
				},
			}),
			// Retain nodes for debugging logs.
			cluster.CreateWithRetain(true),
		)
		if err == nil {
			return nil
		}
	}

	// We failed to create the cluster maxKindTries times, to fail out.
	return err
}

// newKind creates a new Kind cluster for use in testing with the specified name.
//
// Automatically registers the cluster to be deleted at the end of the test.
func newKind(t *testing.T, name, tmpDir string, opts ntopts.KindCluster) (*rest.Config, string) {
	p := cluster.NewProvider()
	kcfgPath := filepath.Join(tmpDir, kubeconfig)

	start := time.Now()
	t.Logf("started creating cluster at %s", start.Format(time.RFC3339))

	err := createKindCluster(p, name, kcfgPath, opts.Version)
	creationSuccessful := err == nil
	finish := time.Now()

	// Register the cluster to be deleted at the end of the test, even if cluster
	// creation failed.
	t.Cleanup(func() {
		if t.Failed() && *e2e.Debug {
			t.Errorf(`Conect to kind cluster:
kind export kubeconfig --name=%s`, name)
			t.Errorf(`Delete kind cluster:
kind delete cluster --name=%s`, name)
			return
		}

		if !creationSuccessful {
			// Since we have set retain=true, the cluster is still available even
			// though creation did not execute successfully.
			artifactsDir := os.Getenv("ARTIFACTS")
			if artifactsDir == "" {
				artifactsDir = filepath.Join(tmpDir, "artifacts")
			}
			t.Logf("exporting failed cluster logs to %s", artifactsDir)
			err := exec.Command("kind", "export", "logs", "--name", name, artifactsDir).Run()
			if err != nil {
				t.Errorf("exporting kind logs: %v", err)
			}
		}

		// If the test runner stops testing with a command like ^C, cleanup
		// callbacks such as this are not executed.
		err := p.Delete(name, kcfgPath)
		if err != nil {
			t.Errorf("deleting Kind cluster %q: %v", name, err)
		}
	})

	if err != nil {
		t.Logf("failed creating cluster at %s", finish.Format(time.RFC3339))
		t.Logf("command took %v to fail", finish.Sub(start))
		t.Fatalf("creating Kind cluster: %v", err)
	}
	t.Logf("finished creating cluster at %s", finish.Format(time.RFC3339))

	// We don't need to specify masterUrl since we have a Kubeconfig.
	cfg, err := clientcmd.BuildConfigFromFlags("", kcfgPath)
	if err != nil {
		t.Fatalf("building rest.Config: %v", err)
	}

	return cfg, kcfgPath
}
