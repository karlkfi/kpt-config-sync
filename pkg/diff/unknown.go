package diff

import (
	"github.com/google/nomos/pkg/core"
)

type unknown struct {
	core.Object
}

var theUnknown = &unknown{}

// Unknown returns a sentinel Object which represents unknown state on the
// cluster. On failing to retrieve the current state of an Object on the
// cluster, a caller should use this to indicate that no action should be
// taken to reconcile the declared version of an Object.
func Unknown() core.Object {
	return theUnknown
}

// IsUnknown returns true if the given Object is the sentinel marker of unknown
// state on the cluster.
func IsUnknown(obj core.Object) bool {
	return obj == theUnknown
}
