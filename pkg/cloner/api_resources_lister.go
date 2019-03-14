package cloner

import (
	"sort"

	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIResourcesLister mimics the smallest required part of discovery.ServerResourcesInterface which
// lists resources available to a Kubernetes cluster.
type APIResourcesLister interface {
	// ServerResources returns the supported API resources for all groups and versions.
	ServerResources() ([]*metav1.APIResourceList, error)
}

// ListResources calls the APIResourcesLister to retrieve the set of APIResources supported
// on the server, and then converts them to a list of GroupVersionResource. Only returns
// APIResources which support the "list" verb.
//
// Currently unused; will be used once `clone` is implemented.
func ListResources(lister APIResourcesLister, errs ErrorAdder) []metav1.APIResource {
	apiResources, err := lister.ServerResources()
	errs.Add(status.APIServerWrapf(err, "unable to list supported API resources"))
	if err != nil {
		return nil
	}

	return flatten(apiResources, errs)
}

// flatten returns a list of the listable APIResources contained in the list of list of
// APIResources. If a given group/resource has multiple versions, returns the most recent.
func flatten(lists []*metav1.APIResourceList, errs ErrorAdder) []metav1.APIResource {
	grs := make(map[schema.GroupKind][]metav1.APIResource)

	for _, list := range lists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		errs.Add(status.APIServerWrapf(err, "error parsing apiVersion"))
		if err != nil {
			// Encountering an error parsing this list doesn't prevent parsing the other lists.
			continue
		}

		for _, resource := range list.APIResources {
			// The server is not guaranteed to populate Group and Version, so this ensures it happens.
			resource.Group = gv.Group
			resource.Version = gv.Version
			gk := schema.GroupKind{
				Group: gv.Group,
				Kind:  resource.Kind,
			}
			grs[gk] = append(grs[gk], resource)
		}
	}

	var result []metav1.APIResource
	for _, resources := range grs {
		sort.Slice(resources, func(i, j int) bool {
			return lessVersions(resources[i], resources[j])
		})
		result = append(result, resources[0])
	}
	return result
}
