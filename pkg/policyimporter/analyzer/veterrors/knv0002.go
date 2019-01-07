package veterrors

import (
	"sort"
	"strings"
)

// DuplicateDirectoryNameErrorCode is the error code for DuplicateDirectoryNameError
const DuplicateDirectoryNameErrorCode = "1002"

func init() {
	register(DuplicateDirectoryNameErrorCode, nil, "")
}

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
type DuplicateDirectoryNameError struct {
	Duplicates []string
}

// Error implements error.
func (e DuplicateDirectoryNameError) Error() string {
	// Ensure deterministic node printing order.
	sort.Strings(e.Duplicates)
	return format(e,
		"Directory names MUST be unique. "+
			"Rename one of these directories:\n\n"+
			"%[1]s",
		strings.Join(e.Duplicates, "\n"))
}

// Code implements Error
func (e DuplicateDirectoryNameError) Code() string { return DuplicateDirectoryNameErrorCode }
