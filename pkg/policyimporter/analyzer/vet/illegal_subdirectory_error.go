package vet

import "github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"

// IllegalSubdirectoryErrorCode is the error code for IllegalSubdirectoryError
const IllegalSubdirectoryErrorCode = "1018"

func init() {
	register(IllegalSubdirectoryErrorCode, nil, "")
}

// IllegalSubdirectoryError reports that the directory has an illegal subdirectory.
type IllegalSubdirectoryError struct {
	BaseDir string
	SubDir  nomospath.Relative
}

// Error implements error
func (e IllegalSubdirectoryError) Error() string {
	return format(e,
		"%s/ directory MUST NOT have subdirectories.\n\n"+
			"path: %[2]s", e.BaseDir, e.SubDir.RelativeSlashPath())
}

// Code implements Error
func (e IllegalSubdirectoryError) Code() string { return IllegalSubdirectoryErrorCode }
