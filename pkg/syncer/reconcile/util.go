package reconcile

import (
	"fmt"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// allVersionNames returns the set of names of all resources with the specified GroupKind.
func allVersionNames(resources map[schema.GroupVersionKind][]*unstructured.Unstructured, gk schema.GroupKind) map[string]bool {
	names := map[string]bool{}
	for gvk, rs := range resources {
		if gvk.GroupKind() != gk {
			continue
		}
		for _, r := range rs {
			n := r.GetName()
			if names[n] {
				panic(fmt.Errorf("duplicate resources names %q declared for %s", n, gvk))
			} else {
				names[n] = true
			}
		}
	}
	return names
}

// cmeForNamespace returns a ConfigManagementError for the given Namespace and error message.
func cmeForNamespace(ns *corev1.Namespace, errMsg string) v1.ConfigManagementError {
	return v1.ConfigManagementError{
		SourcePath:        ns.GetAnnotations()[v1.SourcePathAnnotationKey],
		ResourceName:      ns.GetName(),
		ResourceNamespace: ns.GetNamespace(),
		ResourceGVK:       ns.GroupVersionKind(),
		ErrorMessage:      errMsg,
	}
}

// cmesForResourceError returns ConfigManagementErrors built from the given ResourceError.
func cmesForResourceError(resErr id.ResourceError) []v1.ConfigManagementError {
	resCount := len(resErr.Resources())
	if resCount == 0 {
		return []v1.ConfigManagementError{
			{ErrorMessage: resErr.Error()},
		}
	}

	configErrs := make([]v1.ConfigManagementError, resCount)
	for i, res := range resErr.Resources() {
		configErrs[i] = v1.ConfigManagementError{
			SourcePath:        res.SlashPath(),
			ResourceName:      res.Name(),
			ResourceNamespace: res.Namespace(),
			ResourceGVK:       res.GroupVersionKind(),
			ErrorMessage:      resErr.Error(),
		}
	}
	return configErrs
}
