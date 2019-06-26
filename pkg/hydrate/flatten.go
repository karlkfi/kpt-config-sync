package hydrate

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"k8s.io/apimachinery/pkg/runtime"
)

// Flatten converts an AllConfigs into a list of FileObjects.
func Flatten(c *namespaceconfig.AllConfigs) []runtime.Object {
	var result []runtime.Object
	if c == nil {
		return result
	}

	// Flatten with default filenames.
	if c.CRDClusterConfig != nil {
		for _, crds := range c.CRDClusterConfig.Spec.Resources {
			result = append(result, resourcesToFileObjects(crds)...)
		}
	}
	if c.ClusterConfig != nil {
		for _, clusterObjects := range c.ClusterConfig.Spec.Resources {
			result = append(result, resourcesToFileObjects(clusterObjects)...)
		}
	}
	if c.NamespaceConfigs != nil {
		for _, namespaceConfig := range c.NamespaceConfigs {
			for _, namespaceObjects := range namespaceConfig.Spec.Resources {
				result = append(result, resourcesToFileObjects(namespaceObjects)...)
			}
		}
	}

	return result
}

// resourcesToFileObjects flattens a GenericResources into a list of FileObjects.
func resourcesToFileObjects(r v1.GenericResources) []runtime.Object {
	var result []runtime.Object

	for _, version := range r.Versions {
		for _, raw := range version.Objects {
			result = append(result, raw.Object)
		}
	}

	return result
}
