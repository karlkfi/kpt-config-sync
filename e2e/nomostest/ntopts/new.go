package ntopts

// New is the set of options for instantiating a new NT test.
type New struct {
	// Name is the name of the test. Overrides the one generated from the test
	// name.
	Name string

	// TmpDir is the base temporary directory to use for the test. Overrides the
	// generated directory based on Name and the OS's main temporary directory.
	TmpDir string

	KindCluster
	Nomos
	MultiRepo
}
