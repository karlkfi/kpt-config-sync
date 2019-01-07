package veterrors

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
)

// IllegalLabelDefinitionError represent a set of illegal label definitions.
type IllegalLabelDefinitionError struct {
	ResourceID
	Labels []string
}

// Error implements error.
func (e IllegalLabelDefinitionError) Error() string {
	sort.Strings(e.Labels) // ensure deterministic label order
	l := strings.Join(e.Labels, ", ")
	return format(e,
		"Resources MUST NOT declare labels starting with %[3]q. "+
			"Below Resource declares these offending labels: %[1]s\n\n"+
			"%[2]s",
		l, printResourceID(e), policyhierarchy.GroupName)
}

// Code implements Error
func (e IllegalLabelDefinitionError) Code() string { return IllegalLabelDefinitionErrorCode }
