package status

import (
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/id"
)

type resourceErrorImpl struct {
	underlying Error
	resources  []id.Resource
}

var _ ResourceError = resourceErrorImpl{}

// Error implements error.
func (r resourceErrorImpl) Error() string {
	return format(r)
}

// Is implements Error.
func (r resourceErrorImpl) Is(target error) bool {
	return r.underlying.Is(target)
}

// Code implements Error.
func (r resourceErrorImpl) Code() string {
	return r.underlying.Code()
}

// Body implements Error.
func (r resourceErrorImpl) Body() string {
	return formatBody(r.underlying.Body(), "\n\n", formatResources(r.resources))
}

// Errors implements MultiError.
func (r resourceErrorImpl) Errors() []Error {
	return []Error{r}
}

// Resources implements ResourceError.
func (r resourceErrorImpl) Resources() []id.Resource {
	return r.resources
}

// ToCME implements Error.
func (r resourceErrorImpl) ToCME() v1.ConfigManagementError {
	return fromResourceError(r)
}

// ToCSE implements Error.
func (r resourceErrorImpl) ToCSE() v1alpha1.ConfigSyncError {
	return cseFromResourceError(r)
}

// Cause implements causer
func (r resourceErrorImpl) Cause() error {
	return r.underlying.Cause()
}

// formatResources returns a formatted string containing all Resources in the ResourceError.
func formatResources(resources []id.Resource) string {
	resStrs := make([]string, len(resources))
	for i, res := range resources {
		resStrs[i] = id.PrintResource(res)
	}
	// Sort to ensure deterministic resource printing order.
	sort.Strings(resStrs)
	return strings.Join(resStrs, "\n\n")
}
