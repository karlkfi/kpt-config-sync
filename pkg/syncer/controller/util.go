package controller

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// extractGVKS returns the GroupVersionKind keys in the resourceTypes map.
func extractGVKs(resourceTypes map[schema.GroupVersionKind]runtime.Object) []schema.GroupVersionKind {
	gvks := make([]schema.GroupVersionKind, len(resourceTypes))
	i := 0
	for gvk := range resourceTypes {
		gvks[i] = gvk
		i++
	}
	return gvks
}
