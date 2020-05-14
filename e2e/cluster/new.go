package cluster

import (
	"testing"

	"github.com/google/nomos/e2e"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New establishes a connection to a test cluster.
func New(t *testing.T) *Client {
	t.Helper()

	if !*e2e.E2E {
		// This safeguard prevents test authors from accidentally forgetting to
		// check for the e2e test flag, so `go test ./...` functions as expected
		// without triggering e2e tests.
		t.Fatal(`Attempting to create cluster for non-e2e test. To fix, copy TestMain() from another e2e test.`)
	}

	// TODO(b/1041731): Spin up Kind Cluster.
	//  Register cleanup callback with t.Cleanup(tear down cluster).
	opts := genericclioptions.ConfigFlags{}

	mapper, err := opts.ToRESTMapper()
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := opts.ToRESTConfig()
	if err != nil {
		t.Fatal(err)
	}

	s := runtime.NewScheme()
	err = corev1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}

	c, err := client.New(cfg, client.Options{
		Scheme: s,
		Mapper: mapper,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{Client: c}
}
