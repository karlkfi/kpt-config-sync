package discovery

import (
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// ServerResourcer returns a list of APIResources, or an error if unable to
// retrieve them.
//
// DiscoveryInterface satisfies this interface.
type ServerResourcer interface {
	ServerResources() ([]*metav1.APIResourceList, error)
}

// ClientGetter is a client which can return a CachedDiscoveryInterface.
type ClientGetter interface {
	ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error)
}

// GetResourcesFromClientGetter gets the APIResourceLists from a client's DiscoveryClient.
func GetResourcesFromClientGetter(client ClientGetter) ([]*metav1.APIResourceList, status.MultiError) {
	// Get all known API resources from the server.
	dc, err := client.ToDiscoveryClient()
	if err != nil {
		return nil, status.APIServerError(err, "failed to get discovery client")
	}
	return GetResources(dc)
}

type invalidatable interface {
	Invalidate()
}

// GetResources gets the APIResourceLists from an existing DiscoveryClient.
// Invalidates the cache if possible as the server may have new resources since the client was created.
func GetResources(discoveryClient ServerResourcer) ([]*metav1.APIResourceList, status.MultiError) {
	if invalidatableDiscoveryClient, isInvalidatable := discoveryClient.(invalidatable); isInvalidatable {
		// Non-cached DiscoveryClients aren't invalidatable, so we have to allow for this possibility.
		invalidatableDiscoveryClient.Invalidate()
	}
	resourceLists, discoveryErr := discoveryClient.ServerResources()
	if discoveryErr != nil {
		return nil, status.APIServerError(discoveryErr, "failed to get server resources")
	}
	return resourceLists, nil
}
