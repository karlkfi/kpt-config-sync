package nomostest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

// Use release images from https://github.com/kubernetes-sigs/kind/releases
const kind1_14 = "kindest/node:v1.14.10@sha256:6cd43ff41ae9f02bb46c8f455d5323819aec858b99534a290517ebc181b443c6"

// kubeconfig is the filename of the KUBECONFIG file.
const kubeconfig = "KUBECONFIG"

func createKindCluster(p *cluster.Provider, name, kcfgPath string) error {
	// TODO(willbeason): Allow specifying Kubernetes version.
	return p.Create(name,
		// Use Kubernetes 1.14
		cluster.CreateWithNodeImage(kind1_14),
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
		// Retain nodes for debugging if cluster creation fails.
		cluster.CreateWithRetain(true),
	)
}

// newKind creates a new Kind cluster for use in testing with the specified name.
//
// Automatically registers the cluster to be deleted at the end of the test.
func newKind(t *testing.T, name string, tmpDir string) (*rest.Config, string) {
	// TODO(willbeason): Extract this logic into a CLI that doesn't require
	//  testing.T.
	p := cluster.NewProvider()
	kcfgPath := filepath.Join(tmpDir, kubeconfig)

	start := time.Now()
	t.Logf("started creating cluster at %s", start.Format(time.RFC3339))
	err := createKindCluster(p, name, kcfgPath)
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
