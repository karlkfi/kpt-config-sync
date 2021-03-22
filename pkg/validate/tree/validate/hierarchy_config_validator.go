package validate

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HierarchyConfig verifies that all HierarchyConfig objects specify valid
// namespace-scoped resource kinds and valid inheritance modes.
func HierarchyConfig(tree *objects.Tree) status.MultiError {
	clusterGKs := make(map[schema.GroupKind]bool)
	for _, obj := range tree.Cluster {
		clusterGKs[obj.GetObjectKind().GroupVersionKind().GroupKind()] = true
	}

	var errs status.MultiError
	for _, obj := range tree.HierarchyConfigs {
		errs = status.Append(errs, validateHC(obj, clusterGKs))
	}
	return errs
}

func validateHC(obj ast.FileObject, clusterGKs map[schema.GroupKind]bool) status.MultiError {
	s, err := obj.Structured()
	if err != nil {
		return err
	}

	var errs status.MultiError
	for _, res := range s.(*v1.HierarchyConfig).Spec.Resources {
		// First validate HierarchyMode.
		switch res.HierarchyMode {
		case v1.HierarchyModeNone, v1.HierarchyModeInherit, v1.HierarchyModeDefault:
		default:
			errs = status.Append(errs, hierarchyconfig.IllegalHierarchyModeError(obj, groupKinds(res)[0], res.HierarchyMode))
		}

		// Then validate resource GroupKinds.
		for _, gk := range groupKinds(res) {
			if unsupportedGK(gk) {
				errs = status.Append(errs, hierarchyconfig.UnsupportedResourceInHierarchyConfigError(obj, gk))
			} else if clusterGKs[gk] {
				errs = status.Append(errs, hierarchyconfig.ClusterScopedResourceInHierarchyConfigError(obj, gk))
			}
		}
	}
	return errs
}

func groupKinds(res v1.HierarchyConfigResource) []schema.GroupKind {
	if len(res.Kinds) == 0 {
		return []schema.GroupKind{{Group: res.Group, Kind: ""}}
	}

	gks := make([]schema.GroupKind, len(res.Kinds))
	for i, kind := range res.Kinds {
		gks[i] = schema.GroupKind{Group: res.Group, Kind: kind}
	}
	return gks
}

func unsupportedGK(gk schema.GroupKind) bool {
	return gk == kinds.Namespace().GroupKind() || gk.Group == configmanagement.GroupName || gk.Kind == ""
}
