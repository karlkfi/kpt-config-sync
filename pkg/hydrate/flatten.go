package hydrate

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	corev1 "k8s.io/api/core/v1"
)

// Flatten converts an AllConfigs into a list of FileObjects.
func Flatten(c *namespaceconfig.AllConfigs) []core.Object {
	var result []core.Object
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
		for _, namespaceCfg := range c.NamespaceConfigs {
			// Construct Namespace from NamespaceConfig
			namespace := &corev1.Namespace{}
			namespace.SetGroupVersionKind(kinds.Namespace())
			// Note that this copies references to Annotations/Labels.
			namespace.ObjectMeta = namespaceCfg.ObjectMeta
			result = append(result, namespace)

			for _, namespaceObjects := range namespaceCfg.Spec.Resources {
				result = append(result, resourcesToFileObjects(namespaceObjects)...)
			}
		}
	}

	return result
}

// resourcesToFileObjects flattens a GenericResources into a list of FileObjects.
func resourcesToFileObjects(r v1.GenericResources) []core.Object {
	var result []core.Object

	for _, version := range r.Versions {
		for _, raw := range version.Objects {
			// We assume a GenericResources will only hold KubernetesObjects, and never Lists.
			result = append(result, raw.Object.(core.Object))
		}
	}

	return result
}
