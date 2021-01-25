package ntopts

import (
	"testing"

	"github.com/google/nomos/e2e"
	"k8s.io/client-go/rest"
)

// Opt is an option type for ntopts.New.
type Opt func(opt *New)

// New is the set of options for instantiating a new NT test.
type New struct {
	// Name is the name of the test. Overrides the one generated from the test
	// name.
	Name string

	// TmpDir is the base temporary directory to use for the test. Overrides the
	// generated directory based on Name and the OS's main temporary directory.
	TmpDir string

	// RESTConfig is the config for creating a Client connection to a K8s cluster.
	RESTConfig *rest.Config

	Nomos
	MultiRepo
}

// RequireManual requires the --manual flag is set. Otherwise it will skip the test.
// This avoids running tests (e.g stress tests) that aren't safe to run against a remote cluster automatically.
func RequireManual(t *testing.T) Opt {
	if !*e2e.Manual {
		t.Skip("Must pass --manual so this isn't accidentally run against a test cluster automatically.")
	}
	return func(opt *New) {}
}
