package discovery

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIInfo caches whether APIResources are Namespaced and which are synced.
type APIInfo struct {
	// groupVersionKinds holds the set of known GroupVersionKinds
	groupVersionKinds map[schema.GroupVersionKind]bool
}

// NewAPIInfo returns a new APIInfo object
func NewAPIInfo(resourceLists []*metav1.APIResourceList) (*APIInfo, error) {
	result := &APIInfo{
		groupVersionKinds: map[schema.GroupVersionKind]bool{},
	}

	for _, resourceList := range resourceLists {
		groupVersion, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse discovery APIResourceList")
		}
		for _, resource := range resourceList.APIResources {
			gvk := groupVersion.WithKind(resource.Kind)
			result.groupVersionKinds[gvk] = true
		}
	}

	return result, nil
}

// GroupVersionKinds returns a set of GroupVersionKinds represented by the slice of Syncs with only
// Group and Kind specified.
func (a *APIInfo) GroupVersionKinds(syncs ...*v1.Sync) map[schema.GroupVersionKind]bool {
	syncedGks := make(map[schema.GroupKind]bool, len(syncs))
	for _, sync := range syncs {
		syncedGks[schema.GroupKind{Group: sync.Spec.Group, Kind: sync.Spec.Kind}] = true
	}

	syncedGvks := make(map[schema.GroupVersionKind]bool, len(syncs))
	for gvk := range a.groupVersionKinds {
		if syncedGks[gvk.GroupKind()] {
			syncedGvks[gvk] = true
		}
	}
	return syncedGvks
}
