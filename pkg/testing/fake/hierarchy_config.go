package fake

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HierarchyConfigKind adds a single GVK to a HierarchyConfig.
func HierarchyConfigKind(mode v1.HierarchyModeType, gvk schema.GroupVersionKind) core.MetaMutator {
	return HierarchyConfigResource(mode, gvk.GroupVersion(), gvk.Kind)
}

// HierarchyConfigResource adds a HierarchyConfigResource to a HierarchyConfig.
func HierarchyConfigResource(mode v1.HierarchyModeType, gv schema.GroupVersion, kinds ...string) core.MetaMutator {
	return func(o core.Object) {
		hc := o.(*v1.HierarchyConfig)
		hc.Spec.Resources = append(hc.Spec.Resources,
			v1.HierarchyConfigResource{
				Group:         gv.Group,
				Kinds:         kinds,
				HierarchyMode: mode,
			})
	}
}

// HierarchyConfig initializes  HierarchyConfig in a FileObject.
func HierarchyConfig(opts ...core.MetaMutator) ast.FileObject {
	return HierarchyConfigAtPath("system/hc.yaml", opts...)
}

// HierarchyConfigAtPath returns a HierarchyConfig at the specified path.
func HierarchyConfigAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	result := &v1.HierarchyConfig{TypeMeta: toTypeMeta(kinds.HierarchyConfig())}
	defaultMutate(result)
	for _, opt := range opts {
		opt(result)
	}

	return FileObject(result, path)
}
