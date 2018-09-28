package manager

import (
	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersionKinds returns a set of GroupVersionKind represented by the slice of Syncs.
func GroupVersionKinds(syncs ...*nomosv1alpha1.Sync) map[schema.GroupVersionKind]bool {
	gvks := make(map[schema.GroupVersionKind]bool)
	for _, sync := range syncs {
		for _, g := range sync.Spec.Groups {
			k := g.Kinds
			for _, v := range k.Versions {
				gvk := schema.GroupVersionKind{
					Group:   g.Group,
					Version: v.Version,
					Kind:    k.Kind,
				}
				gvks[gvk] = true
			}
		}
	}
	return gvks
}
