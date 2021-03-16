package selectors

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// ObjectHasUnknownNamespaceSelector reports that `resource`'s namespace-selector annotation
// references a NamespaceSelector that does not exist.
func ObjectHasUnknownNamespaceSelector(resource id.Resource, selector string) status.Error {
	return objectHasUnknownSelector.
		Sprintf("Config %q MUST refer to an existing NamespaceSelector, but has annotation \"%s=%s\" which maps to no declared NamespaceSelector",
			resource.GetName(), v1.NamespaceSelectorAnnotationKey, selector).
		BuildWithResources(resource)
}

// ObjectNotInNamespaceSelectorSubdirectory reports that `resource` is not in a subdirectory of the directory
// declaring `selector`.
func ObjectNotInNamespaceSelectorSubdirectory(resource id.Resource, selector id.Resource) status.Error {
	return objectHasUnknownSelector.
		Sprintf("Config %q MUST refer to a NamespaceSelector in its directory or a parent directory. "+
			"Either remove the annotation \"%s=%s\" from %q or move NamespaceSelector %q to a parent directory of %q.",
			resource.GetName(), v1.NamespaceSelectorAnnotationKey, selector.GetName(), resource.GetName(), selector.GetName(), resource.GetName()).
		BuildWithResources(selector, resource)
}
