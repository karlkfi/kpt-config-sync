package status

import (
	"fmt"
)

// Error defines a Kubernetes Nomos Vet error
// These are GKE Config Management directory errors which are shown to the user and documented.
type Error interface {
	Error() string
	Code() string
}

// Format formats the start of error messages consistently.
func Format(err Error, format string, a ...interface{}) string {
	return fmt.Sprintf("KNV%s: ", err.Code()) + fmt.Sprintf(format, a...)
}

// PathError defines a status error associated with one or more path-identifiable locations in the
// repo.
type PathError interface {
	Error
	RelativePaths() []string
}
