package ntopts

import (
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
