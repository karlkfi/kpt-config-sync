package reconcile

import (
	"fmt"

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
