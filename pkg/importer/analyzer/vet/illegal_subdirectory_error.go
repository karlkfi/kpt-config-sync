package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalSubdirectoryErrorCode is the error code for IllegalSubdirectoryError
const IllegalSubdirectoryErrorCode = "1018"

func init() {
	status.Register(IllegalSubdirectoryErrorCode, IllegalSubdirectoryError{
		BaseDir: "system",
		SubDir:  cmpath.FromSlash("system/foo"),
	})
}

// IllegalSubdirectoryError reports that the directory has an illegal subdirectory.
type IllegalSubdirectoryError struct {
	BaseDir string
	SubDir  cmpath.Path
}

var _ status.PathError = &IllegalSubdirectoryError{}

// Error implements error
func (e IllegalSubdirectoryError) Error() string {
	return status.Format(e,
		"The %s/ directory MUST NOT have subdirectories.", e.BaseDir)
}

// Code implements Error
func (e IllegalSubdirectoryError) Code() string { return IllegalSubdirectoryErrorCode }

// RelativePaths implements PathError
func (e IllegalSubdirectoryError) RelativePaths() []id.Path {
	return []id.Path{e.SubDir}
}
