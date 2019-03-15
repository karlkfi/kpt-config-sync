package importer

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// DynamicResourcer wraps a dynamic.Interface to make it compatible with the Resourcer interface.
type DynamicResourcer struct {
	dynamic.Interface
}

// Resource implements Resourcer.
func (r DynamicResourcer) Resource(resource schema.GroupVersionResource) Lister {
	return r.Interface.Resource(resource)
}
