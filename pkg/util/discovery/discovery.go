package discovery

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type scoperKey struct{}

// AddScoper returns a copy of the Extension with the APIInfo set.
// The value is only accessible with GetScoper.
func AddScoper(r *ast.Root, scoper Scoper) status.Error {
	var err status.Error
	r.Data, err = ast.Add(r.Data, scoperKey{}, scoper)
	return err
}

// GetScoper gets the APIInfo from the Extension.
func GetScoper(r *ast.Root) (Scoper, status.Error) {
	result, err := ast.Get(r.Data, scoperKey{})
	if err != nil {
		return nil, err
	}
	return result.(Scoper), nil
}

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
