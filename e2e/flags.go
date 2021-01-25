// Package e2e defines e2e-test-specific imports and flags for use in e2e
// testing.
package e2e

import (
	"flag"
	"fmt"

	// kubectl auth provider plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// E2E enables running end-to-end tests.
var E2E = flag.Bool("e2e", false,
	"If true, run end-to-end tests. Otherwise do nothing and exit normally.")

// Debug enables running the test in debug mode.
// In debug mode:
// 1) Test execution immediately stops on a call to t.Fatal.
// 2) The test prints the absolute path to the test temporary directory, and
//      not delete it.
// 3) The test prints out how to connect to the kind cluster.
var Debug = flag.Bool("debug", false,
	"If true, do not destroy cluster and clean up temporary directory after test.")

// KubernetesVersion is the version of Kubernetes to test against. Only has effect
// when testing against test-created Kind clusters.
var KubernetesVersion = flag.String("kubernetes-version", "1.17",
	"The version of Kubernetes to create")

// MultiRepo enables running the tests against multi-repo Config Sync.
var MultiRepo = flag.Bool("multirepo", false,
	"If true, configure multi-repo Config Sync. Otherwise configure mono-repo.")

// SkipMode will only run the skipped multi repo tests.
var SkipMode = flag.String("skip-mode", "",
	"Runs tests as given by the mode, one of \"\", runAll, runSkipped to run normally, run all tests, or run only skipped tests respectively")

// DefaultImagePrefix points to the local docker registry.
const DefaultImagePrefix = "localhost:5000"

// ImagePrefix is where the Docker images are stored.
var ImagePrefix = flag.String("image-prefix", DefaultImagePrefix,
	"The prefix to use for Docker images. Defaults to the local Docker registry. Omit the trailing slash. For example, 'gcr.io/stolos-dev/willbeason/1604693819'")

// Manual indicates the test is being run manually. Some tests are not yet safe
// to be run automatically.
var Manual = flag.Bool("manual", false,
	"Specify that the test is being run manually.")

// TestCluster specifies the cluster config used for testing.
var TestCluster = flag.String("test-cluster", Kind,
	fmt.Sprintf("The cluster config used for testing. Allowed values are: %s and %s. "+
		"If --test-cluster=%s, create a Kind cluster. Otherwise use the context specified in %s.",
		Kubeconfig, Kind, Kind, Kubeconfig))

// KubeConfig specifies the file path to the kubeconfig file.
var KubeConfig = flag.String(Kubeconfig, "",
	"The file path to the kubeconfig file. If not set, use the default context.")

const (
	// RunAll runs all tests whether skipped or not
	RunAll = "runAll"
	// RunSkipped runs only skipped tests
	RunSkipped = "runSkipped"
	// RunDefault runs tests as normal and skips skipped tests
	RunDefault = ""
)

const (
	// Kind indicates creating a Kind cluster for testing.
	Kind = "kind"
	// Kubeconfig indicates using a cluster via KUBECONFIG for testing.
	Kubeconfig = "kubeconfig"
)
