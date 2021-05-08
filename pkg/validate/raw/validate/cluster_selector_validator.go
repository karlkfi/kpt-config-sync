package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterSelectorsForHierarchical verifies that all ClusterSelectors have a
// unique name and are under the correct top-level directory. It also verifies
// that no invalid FileObjects are cluster-selected.
func ClusterSelectorsForHierarchical(objs *objects.Raw) status.MultiError {
	return clusterSelectors(objs, true)
}

// ClusterSelectorsForUnstructured verifies that all ClusterSelectors have a
// unique name. It also verifies that no invalid FileObjects are cluster-selected.
func ClusterSelectorsForUnstructured(objs *objects.Raw) status.MultiError {
	return clusterSelectors(objs, false)
}

func clusterSelectors(objs *objects.Raw, checkDir bool) status.MultiError {
	var errs status.MultiError
	clusterGK := kinds.Cluster().GroupKind()
	selectorGK := kinds.ClusterSelector().GroupKind()
	matches := make(map[string][]client.Object)

	for _, obj := range objs.Objects {
		if err := validateClusterSelectorAnnotation(obj); err != nil {
			errs = status.Append(errs, err)
		}

		objGK := obj.GetObjectKind().GroupVersionKind().GroupKind()
		if objGK == clusterGK || objGK == selectorGK {
			if checkDir {
				sourcePath := obj.OSPath()
				dir := cmpath.RelativeSlash(sourcePath).Split()[0]
				if dir != repo.ClusterRegistryDir {
					errs = status.Append(errs, validation.ShouldBeInClusterRegistryError(obj))
				}
			}
			if objGK == selectorGK {
				matches[obj.GetName()] = append(matches[obj.GetName()], obj)
			}
		}
	}

	for name, duplicates := range matches {
		if len(duplicates) > 1 {
			errs = status.Append(errs, nonhierarchical.SelectorMetadataNameCollisionError(selectorGK.Kind, name, duplicates...))
		}
	}

	return errs
}

func validateClusterSelectorAnnotation(obj ast.FileObject) status.Error {
	if !forbidsSelector(obj) {
		return nil
	}

	if _, hasAnnotation := obj.GetAnnotations()[v1.LegacyClusterSelectorAnnotationKey]; hasAnnotation {
		return nonhierarchical.IllegalClusterSelectorAnnotationError(obj, v1.LegacyClusterSelectorAnnotationKey)
	}
	if _, hasAnnotation := obj.GetAnnotations()[v1alpha1.ClusterNameSelectorAnnotationKey]; hasAnnotation {
		return nonhierarchical.IllegalClusterSelectorAnnotationError(obj, v1alpha1.ClusterNameSelectorAnnotationKey)
	}
	return nil
}

func forbidsSelector(obj ast.FileObject) bool {
	gk := obj.GetObjectKind().GroupVersionKind().GroupKind()
	return gk == kinds.Cluster().GroupKind() ||
		gk == kinds.ClusterSelector().GroupKind() ||
		gk == kinds.NamespaceSelector().GroupKind()
}
