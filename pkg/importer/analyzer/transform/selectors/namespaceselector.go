package selectors

import (
	"encoding/json"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/status"
)

// IsConfigApplicableToNamespace returns whether the NamespaceSelector annotation on the given
// config object matches the given labels on a namespace.  The config is applicable if it has no
// such annotation.
func IsConfigApplicableToNamespace(namespaceLabels map[string]string, config core.Object) (bool, status.Error) {
	ls, exists := config.GetAnnotations()[v1.NamespaceSelectorAnnotationKey]
	if !exists {
		return true, nil
	}
	var ns v1.NamespaceSelector
	if err := json.Unmarshal([]byte(ls), &ns); err != nil {
		return false, vet.InvalidSelectorError(config.GetName(), err)
	}
	selector, err := AsPopulatedSelector(&ns.Spec.Selector)
	if err != nil {
		return false, vet.InvalidSelectorError(ns.Name, err)
	}
	return IsSelected(namespaceLabels, selector), nil
}
