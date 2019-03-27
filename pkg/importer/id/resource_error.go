package id

import (
	"strings"

	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// ResourceErrorCode is the error code for a generic ResourceError.
const ResourceErrorCode = "2010"

func init() {
	// TODO: add a way to generate valid error without dependency cycle.
	//status.Register(ResourceErrorCode, resourceError{})
}

// ResourceError defines a status error related to one or more k8s resources.
type ResourceError interface {
	status.Error
	Resources() []Resource
}

// FormatResources returns a formatted string containing all Resources in the ResourceError.
func FormatResources(err ResourceError) string {
	resStrs := make([]string, len(err.Resources()))
	for i, res := range err.Resources() {
		resStrs[i] = PrintResource(res)
	}
	return strings.Join(resStrs, "\n")
}

// resourceError almost always results from an API server call involving one or more resources.
type resourceError struct {
	err       error
	resources []Resource
}

var _ ResourceError = &resourceError{}

// Error implements status.Error
func (r resourceError) Error() string {
	return status.Format(r, "%[1]s\nAffected resources:\n%[2]s",
		r.err.Error(), FormatResources(r))
}

// Code implements status.Error
func (r resourceError) Code() string {
	return ResourceErrorCode
}

// Resources implements ResourceError
func (r resourceError) Resources() []Resource {
	return r.resources
}

// ResourceWrap returns a ResourceError wrapping the given error and Resources.
func ResourceWrap(err error, msg string, resources ...Resource) ResourceError {
	if err == nil {
		return nil
	}
	return resourceError{err: errors.Wrap(err, msg), resources: resources}
}
