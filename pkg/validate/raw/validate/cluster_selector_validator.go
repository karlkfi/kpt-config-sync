package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// ClusterSelectors validates that all ClusterSelectors have a unique name and
// no invalid FileObjects are cluster-selected.
func ClusterSelectors(objs *objects.Raw) status.MultiError {
	var errs status.MultiError
	gk := kinds.ClusterSelector().GroupKind()
	matches := make(map[string][]id.Resource)

	for _, obj := range objs.Objects {
		if err := validateClusterSelectorAnnotation(obj); err != nil {
			errs = status.Append(errs, err)
		}
		if obj.GroupVersionKind().GroupKind() == gk {
			matches[obj.GetName()] = append(matches[obj.GetName()], obj)
		}
	}

	for name, duplicates := range matches {
		if len(duplicates) > 1 {
			errs = status.Append(errs, nonhierarchical.SelectorMetadataNameCollisionError(gk.Kind, name, duplicates...))
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
	gk := obj.GroupVersionKind().GroupKind()
	return gk == kinds.Cluster().GroupKind() ||
		gk == kinds.ClusterSelector().GroupKind() ||
		gk == kinds.NamespaceSelector().GroupKind() ||
		// TODO(b/181135981): Allow ClusterSelectors on CRDs.
		gk == kinds.CustomResourceDefinition()
}
