package ntopts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/docker"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

// KindVersion is a specific Kind version associated with a Kubernetes minor version.
type KindVersion string

// The most recent
const (
	Kind1_18 KindVersion = "kindest/node:v1.18.2@sha256:7b27a6d0f2517ff88ba444025beae41491b016bc6af573ba467b70c5e8e0d85f"
	Kind1_17 KindVersion = "kindest/node:v1.17.5@sha256:ab3f9e6ec5ad8840eeb1f76c89bb7948c77bbf76bcebe1a8b59790b8ae9a283a"
	Kind1_16 KindVersion = "kindest/node:v1.16.9@sha256:7175872357bc85847ec4b1aba46ed1d12fa054c83ac7a8a11f5c268957fd5765"
	Kind1_15 KindVersion = "kindest/node:v1.15.11@sha256:6cc31f3533deb138792db2c7d1ffc36f7456a06f1db5556ad3b6927641016f50"
	Kind1_14 KindVersion = "kindest/node:v1.14.10@sha256:6cd43ff41ae9f02bb46c8f455d5323819aec858b99534a290517ebc181b443c6"
	Kind1_13 KindVersion = "kindest/node:v1.13.12@sha256:214476f1514e47fe3f6f54d0f9e24cfb1e4cda449529791286c7161b7f9c08e7"
	Kind1_12 KindVersion = "kindest/node:v1.12.10@sha256:faeb82453af2f9373447bb63f50bae02b8020968e0889c7fa308e19b348916cb"

	// Kubeconfig is the filename of the KUBECONFIG file.
	Kubeconfig = "KUBECONFIG"

	// maxKindTries is the number of times to attempt to create a Kind cluster for
	// a single test.
	maxKindTries = 6
)

// Kind creates a Kind cluster for the test and fills in the RESTConfig option
// with the information needed to establish a Client with it.
//
// version is one of the KindVersion constants above.
func Kind(t *testing.T, version string) Opt {
	v := asKindVersion(t, version)
	return func(opt *New) {
		opt.RESTConfig = newKind(t, opt.Name, opt.TmpDir, v)
	}
}

// asKindVersion returns the latest Kind version associated with a given
// Kubernetes minor version.
func asKindVersion(t *testing.T, version string) KindVersion {
	t.Helper()

	switch version {
	case "1.12":
		return Kind1_12
	case "1.13":
		return Kind1_13
	case "1.14":
		return Kind1_14
	case "1.15":
		return Kind1_15
	case "1.16":
		return Kind1_16
	case "1.17":
		return Kind1_17
	case "1.18":
		return Kind1_18
	}
	t.Fatalf("Unrecognized Kind version: %q", version)
	return ""
}

// newKind creates a new Kind cluster for use in testing with the specified name.
//
// Automatically registers the cluster to be deleted at the end of the test.
func newKind(t *testing.T, name, tmpDir string, version KindVersion) *rest.Config {
	p := cluster.NewProvider()
	kcfgPath := filepath.Join(tmpDir, Kubeconfig)

	start := time.Now()
	t.Logf("started creating cluster at %s", start.Format(time.RFC3339))

	err := createKindCluster(p, name, kcfgPath, version)
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

	return cfg
}

func createKindCluster(p *cluster.Provider, name, kcfgPath string, version KindVersion) error {
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
  endpoint = ["http://%s:%d"]`, docker.RegistryPort, docker.RegistryName, docker.RegistryPort),
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
