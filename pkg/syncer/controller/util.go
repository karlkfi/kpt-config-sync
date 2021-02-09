package controller

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// extractGVKS returns the GroupVersionKind keys in the resourceTypes map.
func extractGVKs(resourceTypes map[schema.GroupVersionKind]client.Object) []schema.GroupVersionKind {
	gvks := make([]schema.GroupVersionKind, len(resourceTypes))
	i := 0
	for gvk := range resourceTypes {
		gvks[i] = gvk
		i++
	}
	return gvks
}
