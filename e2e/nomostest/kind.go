package nomostest

import (
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
)

// newKind creates a new Kind cluster for use in testing with the specified name.
//
// Automatically registers the cluster to be deleted at the end of the test.
func newKind(t *testing.T, name string, tmpDir string) *rest.Config {
	// TODO(willbeason): Extract this logic into a CLI that doesn't require
	//  testing.T.
	p := cluster.NewProvider()
	kcfgPath := filepath.Join(tmpDir, "KUBECONFIG")

	// TODO(willbeason): Allow specifying Kubernetes version.
	//  Default to 1.16?
	err := p.Create(name, cluster.CreateWithKubeconfigPath(kcfgPath))
	if err != nil {
		t.Fatalf("creating Kind cluster: %v", err)
	}

	// Register the cluster to be deleted at the end of the test.
	t.Cleanup(func() {
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

	return cfg
}
