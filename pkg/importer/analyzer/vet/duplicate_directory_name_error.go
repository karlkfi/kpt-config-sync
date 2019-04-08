package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// DuplicateDirectoryNameErrorCode is the error code for DuplicateDirectoryNameError
const DuplicateDirectoryNameErrorCode = "1002"

func init() {
	status.Register(DuplicateDirectoryNameErrorCode, DuplicateDirectoryNameError{Duplicates: []cmpath.Path{cmpath.FromSlash("namespaces/foo/bar"), cmpath.FromSlash("namespaces/qux/bar")}})
}

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
type DuplicateDirectoryNameError struct {
	Duplicates []cmpath.Path
}

var _ status.PathError = &DuplicateDirectoryNameError{}

// Error implements error.
func (e DuplicateDirectoryNameError) Error() string {
	return status.Format(e,
		"Directory names MUST be unique. Rename one of these directories:")
}

// Code implements Error
func (e DuplicateDirectoryNameError) Code() string { return DuplicateDirectoryNameErrorCode }

// RelativePaths implements PathError
func (e DuplicateDirectoryNameError) RelativePaths() []id.Path {
	paths := make([]id.Path, len(e.Duplicates))
	for i, path := range e.Duplicates {
		paths[i] = path
	}
	return paths
}
