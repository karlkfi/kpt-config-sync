package ntopts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/docker"
	"github.com/google/nomos/e2e/nomostest/testing"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

// KindVersion is a specific Kind version associated with a Kubernetes minor version.
type KindVersion string

// The v0.11.0 images from https://github.com/kubernetes-sigs/kind/releases
const (
	Kind1_21 KindVersion = "kindest/node:v1.21.1@sha256:fae9a58f17f18f06aeac9772ca8b5ac680ebbed985e266f711d936e91d113bad"
	Kind1_20 KindVersion = "kindest/node:v1.20.7@sha256:e645428988191fc824529fd0bb5c94244c12401cf5f5ea3bd875eb0a787f0fe9"
	Kind1_19 KindVersion = "kindest/node:v1.19.11@sha256:7664f21f9cb6ba2264437de0eb3fe99f201db7a3ac72329547ec4373ba5f5911"
	Kind1_18 KindVersion = "kindest/node:v1.18.19@sha256:530378628c7c518503ade70b1df698b5de5585dcdba4f349328d986b8849b1ee"
	Kind1_17 KindVersion = "kindest/node:v1.17.17@sha256:c581fbf67f720f70aaabc74b44c2332cc753df262b6c0bca5d26338492470c17"
	Kind1_16 KindVersion = "kindest/node:v1.16.15@sha256:430c03034cd856c1f1415d3e37faf35a3ea9c5aaa2812117b79e6903d1fc9651"
	Kind1_15 KindVersion = "kindest/node:v1.15.12@sha256:8d575f056493c7778935dd855ded0e95c48cb2fab90825792e8fc9af61536bf9"
	Kind1_14 KindVersion = "kindest/node:v1.14.10@sha256:6033e04bcfca7c5f2a9c4ce77551e1abf385bcd2709932ec2f6a9c8c0aff6d4f"

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
func Kind(t testing.NTB, version string) Opt {
	v := asKindVersion(t, version)
	return func(opt *New) {
		opt.RESTConfig = newKind(t, opt.Name, opt.TmpDir, v)
	}
}

// asKindVersion returns the latest Kind version associated with a given
// Kubernetes minor version.
func asKindVersion(t testing.NTB, version string) KindVersion {
	t.Helper()

	switch version {
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
	case "1.19":
		return Kind1_19
	case "1.20":
		return Kind1_20
	case "1.21":
		return Kind1_21
	}
	t.Fatalf("Unrecognized Kind version: %q", version)
	return ""
}

// newKind creates a new Kind cluster for use in testing with the specified name.
//
// Automatically registers the cluster to be deleted at the end of the test.
func newKind(t testing.NTB, name, tmpDir string, version KindVersion) *rest.Config {
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
			// delete it before retrying.
			err = p.Delete(name, kcfgPath)
			if err != nil {
				return err
			}
		}

		err = p.Create(name,
			// Use the specified version per --kubernetes-version.
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
				// Enable ValidatingAdmissionWebhooks in the Kind cluster, as these
				// are disabled by default.
				KubeadmConfigPatches: []string{
					`
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
apiServer:
  extraArgs:
    "enable-admission-plugins": "ValidatingAdmissionWebhook"`,
				},
			}),
			// Retain nodes for debugging logs.
			cluster.CreateWithRetain(true),
		)
		if err == nil {
			return nil
		}
	}

	// We failed to create the cluster maxKindTries times, so fail out.
	return err
}
