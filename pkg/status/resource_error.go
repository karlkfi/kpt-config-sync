package status

import (
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/pkg/errors"
)

// ResourceErrorCode is the error code for a generic ResourceError.
const ResourceErrorCode = "2010"

func init() {
	Register(ResourceErrorCode, resourceError{})
}

// ResourceError defines a status error related to one or more k8s resources.
type ResourceError interface {
	Error
	Resources() []id.Resource
}

// formatResources returns a formatted string containing all Resources in the ResourceError.
func formatResources(err ResourceError) string {
	resStrs := make([]string, len(err.Resources()))
	for i, res := range err.Resources() {
		resStrs[i] = id.PrintResource(res)
	}
	// Sort to ensure deterministic resource printing order.
	sort.Strings(resStrs)
	return strings.Join(resStrs, "\n\n")
}

// resourceError almost always results from an API server call involving one or more resources.
type resourceError struct {
	err       error
	resources []id.Resource
}

var _ ResourceError = resourceError{}

// Error implements status.Error
func (r resourceError) Error() string {
	return Format(r, "%[1]s\nAffected resources:\n%[2]s",
		r.err.Error(), formatResources(r))
}

// Code implements status.Error
func (r resourceError) Code() string {
	return ResourceErrorCode
}

// Resources implements ResourceError
func (r resourceError) Resources() []id.Resource {
	return r.resources
}

// ResourceWrap returns a ResourceError wrapping the given error and Resources.
func ResourceWrap(err error, msg string, resources ...id.Resource) ResourceError {
	if err == nil {
		return nil
	}
	return resourceError{
		err:       errors.Wrap(err, msg),
		resources: resources,
	}
}

// ToCME implements ToCMEr.
func (r resourceError) ToCME() v1.ConfigManagementError {
	return FromResourceError(r)
}
