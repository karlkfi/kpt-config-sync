package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// DuplicateDirectoryNameErrorCode is the error code for DuplicateDirectoryNameError
const DuplicateDirectoryNameErrorCode = "1002"

func init() {
	status.Register(DuplicateDirectoryNameErrorCode, DuplicateDirectoryNameError{
		Duplicates: []id.Path{
			cmpath.FromSlash("namespaces/foo/bar"),
			cmpath.FromSlash("namespaces/qux/bar")}})
}

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
type DuplicateDirectoryNameError struct {
	Duplicates []id.Path
}

var _ status.PathError = DuplicateDirectoryNameError{}

// Error implements error.
func (e DuplicateDirectoryNameError) Error() string {
	return status.Format(e,
		"Directory names MUST be unique. Rename one of these directories:")
}

// Code implements Error
func (e DuplicateDirectoryNameError) Code() string { return DuplicateDirectoryNameErrorCode }

// RelativePaths implements PathError
func (e DuplicateDirectoryNameError) RelativePaths() []id.Path {
	var paths []id.Path
	copy(paths, e.Duplicates)
	return paths
}

// ToCME implements ToCMEr.
func (e DuplicateDirectoryNameError) ToCME() v1.ConfigManagementError {
	return status.FromPathError(e)
}
