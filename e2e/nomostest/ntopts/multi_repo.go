package ntopts

// MultiRepo configures the NT for use with multi-repo tests.
// If NotRootRepos is non-empty, the test is assumed to be running in
// multi-repo mode.
type MultiRepo struct {
	// NotRootRepos is the (potential) set of repos pointed to by the RepoSyncs.
	//
	// Names are primarily for human-identification and have little functional
	// usage. They don't need to be Namespace or repo-type names, but it makes for
	// easier-to-read tests for the "foo" Namespace to be in the "foo" repository.
	//
	// Each entry must be unique and a valid directory name as we use these to
	// place directories in the test's temporary directory.
	//
	// All are initialized as empty Unstructured repos at the start.
	NotRootRepos []string
}
