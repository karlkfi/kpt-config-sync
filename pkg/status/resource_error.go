package status

import (
	"github.com/google/nomos/pkg/importer/id"
)

// ResourceErrorCode is the error code for a generic ResourceError.
const ResourceErrorCode = "2010"

// ResourceError defines a status error related to one or more k8s resources.
type ResourceError interface {
	Error
	Resources() []id.Resource
}

// resourceError almost always results from an API server call involving one or more resources.
var resourceError = NewErrorBuilder(ResourceErrorCode)

// ResourceWrap returns a ResourceError wrapping the given error and Resources.
func ResourceWrap(err error, msg string, resources ...id.Resource) Error {
	if err == nil {
		return nil
	}
	return resourceError.Sprint(msg).Wrap(err).BuildWithResources(resources...)
}
