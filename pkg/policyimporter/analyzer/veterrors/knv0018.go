package veterrors

// IllegalSubdirectoryError reports that the directory has an illegal subdirectory.
type IllegalSubdirectoryError struct {
	BaseDir string
	SubDir  string
}

// Error implements error
func (e IllegalSubdirectoryError) Error() string {
	return format(e,
		"%s/ directory MUST NOT have subdirectories.\n\n"+
			"path: %[2]s", e.BaseDir, e.SubDir)
}

// Code implements Error
func (e IllegalSubdirectoryError) Code() string { return IllegalSubdirectoryErrorCode }
