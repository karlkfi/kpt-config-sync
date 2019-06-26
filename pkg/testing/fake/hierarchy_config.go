package fake

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HierarchyConfigMutator modifies a HierarchyConfig.
type HierarchyConfigMutator func(hc *v1.HierarchyConfig)

// HierarchyConfigMeta wraps MetaMutators to be specific to HierarchyConfigs.
func HierarchyConfigMeta(opts ...object.MetaMutator) HierarchyConfigMutator {
	return func(hc *v1.HierarchyConfig) {
		mutate(hc, opts...)
	}
}

// HierarchyConfigResource adds a HierarchyConfigResource to a HierarchyConfig.
func HierarchyConfigResource(gvk schema.GroupVersionKind, mode v1.HierarchyModeType) HierarchyConfigMutator {
	return func(hc *v1.HierarchyConfig) {
		hc.Spec.Resources = append(hc.Spec.Resources,
			v1.HierarchyConfigResource{
				Group:         gvk.Group,
				Kinds:         []string{gvk.Kind},
				HierarchyMode: mode,
			})
	}
}

// HierarchyConfig initializes  HierarchyConfig in a FileObject.
func HierarchyConfig(opts ...HierarchyConfigMutator) ast.FileObject {
	return HierarchyConfigAtPath("system/hc.yaml", opts...)
}

// HierarchyConfigAtPath returns a HierarchyConfig at the specified path.
func HierarchyConfigAtPath(path string, opts ...HierarchyConfigMutator) ast.FileObject {
	result := &v1.HierarchyConfig{TypeMeta: toTypeMeta(kinds.HierarchyConfig())}
	defaultMutate(result)
	for _, opt := range opts {
		opt(result)
	}

	return FileObject(result, path)
}
