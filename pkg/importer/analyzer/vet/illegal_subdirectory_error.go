package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// IllegalSubdirectoryErrorCode is the error code for IllegalSubdirectoryError
const IllegalSubdirectoryErrorCode = "1018"

func init() {
	status.AddExamples(IllegalSubdirectoryErrorCode,
		IllegalSubdirectoryError("system", cmpath.FromSlash("system/foo")))
}

var illegalSubdirectoryError = status.NewErrorBuilder(IllegalSubdirectoryErrorCode)

// IllegalSubdirectoryError reports that the directory has an illegal subdirectory.
func IllegalSubdirectoryError(baseDir string, subDir cmpath.Path) status.Error {
	return illegalSubdirectoryError.WithPaths(subDir).Errorf(
		"The %s/ directory MUST NOT have subdirectories.", baseDir)
}
