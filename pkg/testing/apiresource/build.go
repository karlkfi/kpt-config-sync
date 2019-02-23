package apiresource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Opt modifies an APIResource.
type Opt func(r *metav1.APIResource)

// Build constructs an APIResource with the given GroupVersionResource, and modifies it with opts.
func Build(base metav1.APIResource, opts ...Opt) metav1.APIResource {
	result := base
	for _, opt := range opts {
		opt(&result)
	}
	return result
}

// Lists instantiates a []*APIResourceList from a list of APIResources. Splits APIResources into
// lists by GroupVersion.
func Lists(resources ...metav1.APIResource) []*metav1.APIResourceList {
	lists := make(map[schema.GroupVersion]metav1.APIResourceList)

	for _, resource := range resources {
		gv := schema.GroupVersion{Group: resource.Group, Version: resource.Version}
		list := lists[gv]
		list.GroupVersion = gv.String()

		// Unset Group and Version as Kubernetes inconsistently fills in these fields, so we can't rely
		// on them being set. Consumers should rely on APIResourceList.GroupVersion.
		resource.Group = ""
		resource.Version = ""

		list.APIResources = append(list.APIResources, resource)
		lists[gv] = list
	}

	var result []*metav1.APIResourceList
	for i := range lists {
		list := lists[i]
		result = append(result, &list)
	}
	return result
}
