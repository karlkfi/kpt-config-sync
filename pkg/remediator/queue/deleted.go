package queue

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
)

type deleted struct {
	core.Object
}

// MarkDeleted marks the given Object as having been deleted from the cluster.
// On receiving a Deleted event, Watchers should call this *first* and then pass
// the returned Object to Add().
func MarkDeleted(obj core.Object) core.Object {
	if obj == nil {
		glog.Warning("Attempting to mark nil object as deleted")
		// TODO(b/162601559): Increment internal error metric here
		return obj
	}
	return &deleted{obj}
}

// WasDeleted returns true if the given Object was marked as having been
// deleted from the cluster.
func WasDeleted(obj core.Object) bool {
	if obj == nil {
		glog.Warning("Attempting to check nil object for WasDeleted")
		// TODO(b/162601559): Increment internal error metric here
		return false
	}
	_, wasDeleted := obj.(*deleted)
	return wasDeleted
}
