package discovery

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectScope is the return type for APIInfo.GetScope
type ObjectScope string

const (
	// ClusterScope is an object scoped to the cluster
	ClusterScope = ObjectScope("cluster")
	// NamespaceScope is an object scoped to namespace
	NamespaceScope = ObjectScope("namespace")
	// UnknownScope is returned if the object does not exist in APIInfo
	UnknownScope = ObjectScope("unknown")
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

	// groupKindsNamespaced is true for Namespaced GroupKinds, false if not Namespaced, and not
	// present if missing.
	groupKindsNamespaced map[schema.GroupKind]bool
}

// NewAPIInfo returns a new APIInfo object
func NewAPIInfo(resourceLists []*metav1.APIResourceList) (*APIInfo, error) {
	result := &APIInfo{
		groupVersionKinds:    map[schema.GroupVersionKind]bool{},
		groupKindsNamespaced: map[schema.GroupKind]bool{},
	}
	for _, resourceList := range resourceLists {
		groupVersion, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse discovery APIResourceList")
		}
		for _, resource := range resourceList.APIResources {
			gvk := groupVersion.WithKind(resource.Kind)
			result.groupKindsNamespaced[gvk.GroupKind()] = resource.Namespaced
			result.groupVersionKinds[gvk] = true
		}
	}

	return result, nil
}

// AddCustomResources updates APIInfo with custom resource metadata from the provided CustomResourceDefinitions.
// It does not replace anything that already exists in APIInfo.
func (a *APIInfo) AddCustomResources(crds ...*v1beta1.CustomResourceDefinition) {
	for _, crd := range crds {
		crSpec := crd.Spec

		gk := schema.GroupKind{Group: crSpec.Group, Kind: crSpec.Names.Kind}
		// CRD Scope defaults to Namespaced
		namespaced := crSpec.Scope != v1beta1.ClusterScoped
		for _, v := range crSpec.Versions {
			if !v.Served {
				continue
			}
			gvk := gk.WithVersion(v.Name)
			if _, found := a.groupVersionKinds[gvk]; found {
				continue
			}
			a.groupVersionKinds[gvk] = true
			a.groupKindsNamespaced[gk] = namespaced
		}

		if version := crSpec.Version; version != "" {
			// For compatibility with deprecated Version field.
			gvk := gk.WithVersion(version)
			if _, found := a.groupVersionKinds[gvk]; found {
				continue
			}
			a.groupVersionKinds[gvk] = true
			a.groupKindsNamespaced[gk] = namespaced
		}
	}
}

// GetScope returns the scope for the GroupKind, or UnknownScope if not found.
func (a *APIInfo) GetScope(gk schema.GroupKind) ObjectScope {
	namespaced, found := a.groupKindsNamespaced[gk]
	if !found {
		return UnknownScope
	}
	if namespaced {
		return NamespaceScope
	}
	return ClusterScope
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
