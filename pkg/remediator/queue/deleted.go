package queue

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metrics"
	"k8s.io/apimachinery/pkg/runtime"
)

type deleted struct {
	core.Object
}

// DeepCopyObject implements client.Object.
//
// This ensures when we deepcopy a deleted object, we retain that it is deleted.
func (d *deleted) DeepCopyObject() runtime.Object {
	return &deleted{Object: d.Object.DeepCopyObject().(core.Object)}
}

// MarkDeleted marks the given Object as having been deleted from the cluster.
// On receiving a Deleted event, Watchers should call this *first* and then pass
// the returned Object to Add().
func MarkDeleted(ctx context.Context, obj core.Object) core.Object {
	if obj == nil {
		glog.Warning("Attempting to mark nil object as deleted")
		metrics.RecordInternalError(ctx, "remediator")
		return obj
	}
	return &deleted{obj}
}

// WasDeleted returns true if the given Object was marked as having been
// deleted from the cluster.
func WasDeleted(ctx context.Context, obj core.Object) bool {
	if obj == nil {
		glog.Warning("Attempting to check nil object for WasDeleted")
		metrics.RecordInternalError(ctx, "remediator")
		return false
	}
	_, wasDeleted := obj.(*deleted)
	return wasDeleted
}
