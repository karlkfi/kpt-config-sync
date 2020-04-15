package reconcile

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/util/openapi"
)

// OpenAPIResources gets the openapi.Resources available on an API Server.
func OpenAPIResources(cfg *rest.Config) (openapi.Resources, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return openapi.NewOpenAPIGetter(dc).Get()
}
