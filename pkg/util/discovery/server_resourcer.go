package discovery

import (
	"github.com/golang/glog"
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
		// Apparently the ServerResources batches a bunch of discovery requests calls
		// and the author decided that it's perfectly reasonable to return an error
		// for failure on any of those calls (despite some succeeding), so we
		// check for this specific error then ignore it while logging a warning.
		// It's not clear how we should handle this error since there's not a good
		// way to determine if we really needed the discovery info from that one
		// group that failed and something is going horribly wrong, or if someone
		// decided to have fun with adding broken APIServices.  In any case, this is
		// Kubernetes so we are going to continue onward in the name of eventual
		// consistency, tally-ho!
		if discovery.IsGroupDiscoveryFailedError(discoveryErr) {
			glog.Warningf("failed to discover some APIGroups: %s", discoveryErr)
		} else {
			return nil, status.APIServerError(discoveryErr, "failed to get server resources")
		}
	}
	return resourceLists, nil
}
