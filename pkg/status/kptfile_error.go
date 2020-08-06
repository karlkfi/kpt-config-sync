package status

import (
	"strings"
)

// MultipleKptfilesErrorCode is the error code for MultipleKptfilesError
const MultipleKptfilesErrorCode = "1059"

var multipleKptfilesError = NewErrorBuilder(MultipleKptfilesErrorCode)

// MultipleKptfilesError reports that there are multiple Kptfiles in a repo.
func MultipleKptfilesError(paths ...string) Error {
	return multipleKptfilesError.
		Sprintf("Repo must contain at most one Kptfile:\n%s",
			strings.Join(paths, "\n")).
		Build()
}
