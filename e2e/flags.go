// Package e2e
package e2e

import (
	"flag"

	// kubectl auth provider plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// E2E enables running end-to-end tests.
var E2E = flag.Bool("e2e", false,
	"If true, run end-to-end tests. Otherwise do nothing and exit normally.")
