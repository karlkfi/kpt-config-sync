package fake

import (
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// discoveryClient implements the subset of the DiscoveryInterface used by the
// Syncer.
type discoveryClient struct {
	resources []*metav1.APIResourceList
}

// ServerResources implements discovery.ServerResourcer.
func (d discoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return d.resources, nil
}

var _ discovery.ServerResourcer = discoveryClient{}

// NewDiscoveryClient returns a discoveryClient that reports types available
// to the API Server.
//
// Does not report the scope of each GVK as no tests requiring a discoveryClient
// use scope information.
func NewDiscoveryClient(gvks ...schema.GroupVersionKind) discovery.ServerResourcer {
	gvs := make(map[string][]string)
	for _, gvk := range gvks {
		gv := gvk.GroupVersion().String()
		if _, found := gvs[gv]; !found {
			gvs[gv] = []string{}
		}
		gvs[gv] = append(gvs[gv], gvk.Kind)
	}

	var resources []*metav1.APIResourceList
	for gv, kinds := range gvs {
		resource := &metav1.APIResourceList{
			GroupVersion: gv,
		}
		for _, k := range kinds {
			resource.APIResources = append(resource.APIResources,
				metav1.APIResource{
					Kind: k,
				})
		}
	}

	return discoveryClient{
		resources: resources,
	}
}
