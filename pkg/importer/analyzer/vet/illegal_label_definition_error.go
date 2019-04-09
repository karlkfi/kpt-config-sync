package vet

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalLabelDefinitionErrorCode is the error code for IllegalLabelDefinitionError
const IllegalLabelDefinitionErrorCode = "1011"

func init() {
	status.Register(IllegalLabelDefinitionErrorCode, IllegalLabelDefinitionError{
		Resource: role(),
		Labels:   []string{v1.SyncTokenAnnotationKey, v1.ResourceManagementKey},
	})
}

// IllegalLabelDefinitionError represent a set of illegal label definitions.
type IllegalLabelDefinitionError struct {
	id.Resource
	Labels []string
}

var _ status.ResourceError = &IllegalLabelDefinitionError{}

// Error implements error.
func (e IllegalLabelDefinitionError) Error() string {
	labels := e.Labels
	sort.Strings(labels) // ensure deterministic label order
	labels2 := make([]string, len(labels))
	for i, label := range labels {
		labels2[i] = fmt.Sprintf("%q", label)
	}
	l := strings.Join(labels2, ", ")
	return status.Format(e,
		"Resources MUST NOT declare labels starting with %q. "+
			"Below Resource declares these offending labels: %s",
		v1.ConfigManagementPrefix, l)
}

// Code implements Error
func (e IllegalLabelDefinitionError) Code() string { return IllegalLabelDefinitionErrorCode }

// Resources implements ResourceError
func (e IllegalLabelDefinitionError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
