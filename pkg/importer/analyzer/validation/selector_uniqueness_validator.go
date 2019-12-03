package validation

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// selectorUniquenessValidator ensures ClusterSelectors and NamespaceSelectors
// have globally-unique names in the repository. Otherwise, it is ambiguous
// which one to use.
type selectorUniquenessValidator struct {
	selectorGVK schema.GroupVersionKind
}

// ClusterSelectorUniqueness ensures no two ClusterSelectors have the
// same metadata.name.
//
// This should be done before resolving ClusterSelectors.
// We can't check for NamespaceSelector uniqueness at the same time, as we can't
// be sure whether NamespaceSelectors are duplicates without running the
// Cluster-selection logic.
var ClusterSelectorUniqueness nonhierarchical.Validator = selectorUniquenessValidator{selectorGVK: kinds.ClusterSelector()}

// NamespaceSelectorUniqueness ensures no two NamespaceSelectors have
// the same metadata.name.
//
// This should be run *after* ClusterSelection is resolved, as we allow
// NamespaceSelectors to define the ClusterSelector annotation.
var NamespaceSelectorUniqueness nonhierarchical.Validator = selectorUniquenessValidator{selectorGVK: kinds.NamespaceSelector()}

// Validate implements Validator.
func (v selectorUniquenessValidator) Validate(objects []ast.FileObject) status.MultiError {
	// Collect duplicates with the same name.
	selectors := make(map[string][]id.Resource)
	for _, o := range objects {
		if o.GroupVersionKind() == v.selectorGVK {
			selectors[o.GetName()] = append(selectors[o.GetName()], o)
		}
	}

	var errs status.MultiError
	for name, duplicates := range selectors {
		if len(duplicates) > 1 {
			errs = status.Append(errs, nonhierarchical.SelectorMetadataNameCollisionError(
				v.selectorGVK.Kind, name, duplicates...))
		}
	}
	return errs
}
