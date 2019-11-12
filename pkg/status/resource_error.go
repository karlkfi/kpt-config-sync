package status

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/importer/id"
)

// ResourceErrorCode is the error code for a generic ResourceError.
const ResourceErrorCode = "2010"

// ResourceError defines a status error related to one or more k8s resources.
type ResourceError interface {
	Error
	Resources() []id.Resource
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

// resourceError almost always results from an API server call involving one or more resources.
var resourceError = NewErrorBuilder(ResourceErrorCode)

// ResourceWrap returns a ResourceError wrapping the given error and Resources.
func ResourceWrap(err error, msg string, resources ...id.Resource) Error {
	if err == nil {
		return nil
	}
	return resourceError.WithResources(resources...).Wrap(err, msg)
}
