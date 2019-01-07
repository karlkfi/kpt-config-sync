package veterrors

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
)

// IllegalAnnotationDefinitionErrorCode is the error code for IllegalAnnotationDefinitionError
const IllegalAnnotationDefinitionErrorCode = "1010"

func init() {
	register(IllegalAnnotationDefinitionErrorCode, nil, "")
}

// IllegalAnnotationDefinitionError represents a set of illegal annotation definitions.
type IllegalAnnotationDefinitionError struct {
	ResourceID
	Annotations []string
}

// Error implements error.
func (e IllegalAnnotationDefinitionError) Error() string {
	annotations := e.Annotations
	sort.Strings(annotations) // ensure deterministic annotation order
	annotations2 := make([]string, len(annotations))
	for i, annotation := range annotations {
		annotations2[i] = fmt.Sprintf("%q", annotation)
	}
	a := strings.Join(annotations2, ", ")
	return format(e,
		"Resources MUST NOT declare unsupported annotations starting with %[3]q. "+
			"Resource has offending annotations: %[1]s\n\n"+
			"%[2]s",
		a, printResourceID(e), policyhierarchy.GroupName)
}

// Code implements Error
func (e IllegalAnnotationDefinitionError) Code() string { return IllegalAnnotationDefinitionErrorCode }
