package diff

import "sigs.k8s.io/controller-runtime/pkg/client"

type unknown struct {
	client.Object
}

var theUnknown = &unknown{}

// Unknown returns a sentinel Object which represents unknown state on the
// cluster. On failing to retrieve the current state of an Object on the
// cluster, a caller should use this to indicate that no action should be
// taken to reconcile the declared version of an Object.
func Unknown() client.Object {
	return theUnknown
}

// IsUnknown returns true if the given Object is the sentinel marker of unknown
// state on the cluster.
func IsUnknown(obj client.Object) bool {
	return obj == theUnknown
}
