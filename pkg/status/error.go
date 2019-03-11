package status

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/id"
)

// Error defines a Kubernetes Nomos Vet error
// These are GKE Config Management directory errors which are shown to the user and documented.
type Error interface {
	Error() string
	Code() string
}

// Format formats the start of error messages consistently.
func Format(err Error, format string, a ...interface{}) string {
	switch e := err.(type) {
	case PathError:
		return fmt.Sprintf("KNV%s: %s\nPotential causes: %s",
			e.Code(),
			fmt.Sprintf(format, a...),
			strings.Join(e.RelativePaths(), ", "))
	case ResourceError:
		resStrs := make([]string, len(e.Resources()))
		for i, res := range e.Resources() {
			resStrs[i] = id.PrintResource(res)
		}
		return fmt.Sprintf("KNV%s: %s\nPotential causes: %s",
			e.Code(),
			fmt.Sprintf(format, a...),
			strings.Join(resStrs, "\n"))
	default:
		return fmt.Sprintf("KNV%s: ", e.Code()) + fmt.Sprintf(format, a...)
	}
}

// PathError defines a status error associated with one or more path-identifiable locations in the
// repo.
type PathError interface {
	Error
	RelativePaths() []string
}

// ResourceError defines a status error related to one or more k8s resources.
type ResourceError interface {
	Error
	Resources() []id.Resource
}
