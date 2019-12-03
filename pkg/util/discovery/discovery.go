package discovery

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIInfo caches the GroupVersionKinds available on the cluster.
type APIInfo map[schema.GroupVersionKind]bool

// NewAPIInfo constructs an APIInfo from resources retrieved from a DiscoveryClient, and
// a set of additional resources with known scopes.
//
// Note that this EXCLUDES types defined in CRDs, but are not on the APIServer.
// This is because trying to sync CRs for a CRD that is not yet available to the
// cluster is an error.
func NewAPIInfo(resourceLists []*metav1.APIResourceList) (APIInfo, status.MultiError) {
	gvks, errs := toGVKs(resourceLists...)
	if errs != nil {
		return nil, errs
	}

	result := APIInfo{}
	for _, gvk := range gvks {
		result[gvk] = true
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
	for gvk := range *a {
		if syncedGks[gvk.GroupKind()] {
			syncedGvks[gvk] = true
		}
	}
	return syncedGvks
}

func toGVKs(lists ...*metav1.APIResourceList) ([]schema.GroupVersionKind, status.MultiError) {
	var result []schema.GroupVersionKind
	var errs status.MultiError
	for _, list := range lists {
		groupVersion, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			errs = status.Append(errs, errors.Wrapf(err, "discovery client returned invalid GroupVersion: %q", list.GroupVersion))
			continue
		}
		for _, resource := range list.APIResources {
			result = append(result, groupVersion.WithKind(resource.Kind))
		}
	}

	return result, errs
}
