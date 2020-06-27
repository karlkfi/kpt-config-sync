package nomostest

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"testing"

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

	// Kind seems to allow a max cluster name length of 49.  If we exceed that, hash the
	// name, truncate to 40 chars then append 8 hash digits (32 bits).
	const nameLimit = 49
	const hashChars = 8
	if nameLimit < len(name) {
		hashBytes := sha1.Sum([]byte(name))
		hashStr := hex.EncodeToString(hashBytes[:])
		name = fmt.Sprintf("%s-%s", name[:nameLimit-1-hashChars], hashStr[:hashChars])
	}

	err := createKindCluster(p, name, kcfgPath)

	if err != nil {
		t.Fatalf("creating Kind cluster: %v", err)
	}

	// Register the cluster to be deleted at the end of the test.
	t.Cleanup(func() {
		if t.Failed() && *e2e.Debug {
			t.Errorf(`Conect to kind cluster:
kind export kubeconfig --name=%s`, name)
			t.Errorf(`Delete kind cluster:
kind delete cluster --name=%s`, name)
			return
		}
		// If the test runner stops testing with a command like ^C, cleanup
		// callbacks such as this are not executed.
		err := p.Delete(name, kcfgPath)
		if err != nil {
			t.Errorf("deleting Kind cluster %q: %v", name, err)
		}
	})

	// We don't need to specify masterUrl since we have a Kubeconfig.
	cfg, err := clientcmd.BuildConfigFromFlags("", kcfgPath)
	if err != nil {
		t.Fatalf("building rest.Config: %v", err)
	}

	return cfg, kcfgPath
}
