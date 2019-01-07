package veterrors

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
)

// IllegalAnnotationDefinitionError represents a set of illegal annotation definitions.
type IllegalAnnotationDefinitionError struct {
	ResourceID
	Annotations []string
}

// Error implements error.
func (e IllegalAnnotationDefinitionError) Error() string {
	sort.Strings(e.Annotations) // ensure deterministic annotation order
	a := strings.Join(e.Annotations, ", ")
	return format(e,
		"Resources MUST NOT declare unsupported annotations starting with %[3]q. "+
			"Resource has offending annotations: %[1]s\n\n"+
			"%[2]s",
		a, printResourceID(e), policyhierarchy.GroupName)
}

// Code implements Error
func (e IllegalAnnotationDefinitionError) Code() string { return IllegalAnnotationDefinitionErrorCode }
