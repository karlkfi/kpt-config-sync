package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// DuplicateDirectoryNameErrorCode is the error code for DuplicateDirectoryNameError
const DuplicateDirectoryNameErrorCode = "1002"

func init() {
	status.AddExamples(DuplicateDirectoryNameErrorCode, DuplicateDirectoryNameError(
		cmpath.FromSlash("namespaces/foo/bar"),
		cmpath.FromSlash("namespaces/qux/bar")))
}

var duplicateDirectoryNameError = status.NewErrorBuilder(DuplicateDirectoryNameErrorCode)

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
func DuplicateDirectoryNameError(dirs ...id.Path) status.Error {
	return duplicateDirectoryNameError.WithPaths(dirs...).New(
		"Directory names MUST be unique. Rename one of these directories:")
}
