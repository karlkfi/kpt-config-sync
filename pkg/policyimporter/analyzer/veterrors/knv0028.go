package veterrors

import "path"

// InvalidDirectoryNameErrorCode is the error code for InvalidDirectoryNameError
const InvalidDirectoryNameErrorCode = "1028"

func init() {
	register(InvalidDirectoryNameErrorCode, nil, "")
}

// InvalidDirectoryNameError represents an illegal usage of a reserved name.
type InvalidDirectoryNameError struct {
	Dir string
}

// Error implements error.
func (e InvalidDirectoryNameError) Error() string {
	return format(e,
		"Directories MUST be a valid RFC1123 DNS label. Rename or remove directory:\n\n"+
			"path: %[1]s\n"+
			"name: %[2]s",
		e.Dir, path.Base(e.Dir))
}

// Code implements Error
func (e InvalidDirectoryNameError) Code() string { return InvalidDirectoryNameErrorCode }
