package test

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	apis "github.com/google/nomos/pkg/api/policyascode"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// TestMain runs the integration tests.
// nolint
func TestMain(m *testing.M, cfg **rest.Config) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "manifests", "bespin", "config", "crds")},
	}
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal(err)
	}
	if _, err := t.Start(); err != nil {
		log.Fatal(err)
	}
	*cfg = t.Config
	code := m.Run()
	if err := t.Stop(); err != nil {
		log.Printf("unable to stop test runner: %v", err)
	}
	os.Exit(code)
}
