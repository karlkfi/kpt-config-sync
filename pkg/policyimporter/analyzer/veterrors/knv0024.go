package veterrors

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/id"
)

// IllegalKindInSystemErrorCode is the error code for IllegalKindInSystemError
const IllegalKindInSystemErrorCode = "1024"

func init() {
	register(IllegalKindInSystemErrorCode, nil, "")
}

// IllegalKindInSystemError reports that an object has been illegally defined in system/
type IllegalKindInSystemError struct {
	id.Resource
}

// Error implements error
func (e IllegalKindInSystemError) Error() string {
	return format(e,
		"Resources of this Kind may not be declared in %[2]s/:\n\n"+
			"%[1]s",
		id.PrintResource(e), repo.SystemDir, e.RelativeSlashPath)
}

// Code implements Error
func (e IllegalKindInSystemError) Code() string {
	return IllegalKindInSystemErrorCode
}
