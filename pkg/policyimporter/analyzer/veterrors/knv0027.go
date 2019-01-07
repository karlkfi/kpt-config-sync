package veterrors

// UnsupportedRepoSpecVersion reports that the repo version is not supported.
type UnsupportedRepoSpecVersion struct {
	ResourceID
	Version string
}

// Error implements error
func (e UnsupportedRepoSpecVersion) Error() string {
	return format(e,
		"Unsupported Repo spec.version: %[2]q. Must use version \"0.1.0\"\n\n"+
			"%[1]s",
		printResourceID(e), e.Version)
}

// Code implements Error
func (e UnsupportedRepoSpecVersion) Code() string { return UnsupportedRepoSpecVersionCode }
