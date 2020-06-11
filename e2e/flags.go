// Package e2e defines e2e-test-specific imports and flags for use in e2e
// testing.
package e2e

import (
	"flag"

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
