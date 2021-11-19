package ntopts

import (
	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/testing"
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

	// SkipAutopilot will skip the test if running on an Autopilot cluster.
	SkipAutopilot bool

	Nomos
	MultiRepo
	TestType
}

// RequireManual requires the --manual flag is set. Otherwise it will skip the test.
// This avoids running tests (e.g stress tests) that aren't safe to run against a remote cluster automatically.
func RequireManual(t testing.NTB) Opt {
	if !*e2e.Manual {
		t.Skip("Must pass --manual so this isn't accidentally run against a test cluster automatically.")
	}
	return func(opt *New) {}
}

// SkipAutopilotCluster will skip the test on the autopilot cluster.
func SkipAutopilotCluster(opt *New) {
	opt.SkipAutopilot = true
}
