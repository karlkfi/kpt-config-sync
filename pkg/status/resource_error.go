package status

import "sigs.k8s.io/controller-runtime/pkg/client"

// ResourceErrorCode is the error code for a generic ResourceError.
const ResourceErrorCode = "2010"

// ResourceError defines a status error related to one or more k8s resources.
type ResourceError interface {
	Error
	Resources() []client.Object
}

// ResourceErrorBuilder almost always results from an API server call involving one or more resources.
var ResourceErrorBuilder = NewErrorBuilder(ResourceErrorCode)

// ResourceWrap returns a ResourceError wrapping the given error and Resources.
func ResourceWrap(err error, msg string, resources ...client.Object) Error {
	if err == nil {
		return nil
	}
	return ResourceErrorBuilder.Sprint(msg).Wrap(err).BuildWithResources(resources...)
}
