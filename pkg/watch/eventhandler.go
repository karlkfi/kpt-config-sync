package watch

import (
	"fmt"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// GVKNN adds Version to core.ID to make it suitable for getting an object from
// a cluster into an *unstructured.Unstructured.
type GVKNN struct {
	core.ID
	Version string
}

// GroupVersionKind returns the GVK contained in this GVKNN.
func (gvknn GVKNN) GroupVersionKind() schema.GroupVersionKind {
	return gvknn.GroupKind.WithVersion(gvknn.Version)
}

// queue is the interface to add an event to the work queue
type queue interface {
	Add(gvknn GVKNN)
}

// handler is the event handler that is added to an informer.
// It implements the cache.ResourceEventHandler interface.
// For each event, it will be put into the queue when the
// object contained in the event meets one of the two conditions.
// - The object can be found in the DeclaredResources.
// - The object is being managed by Config Sync.
// Any event that doesn't meet the two conditions are ignored.
type handler struct {
	resources *declaredresources.DeclaredResources
	queue     queue
}

var _ cache.ResourceEventHandler = handler{}

// OnAdd adds a gvknn to the work queue
// by filtering the event.
func (r handler) OnAdd(obj interface{}) {
	r.filterEvent(obj, "add")
}

// OnUpdate adds a gvknn to the work queue
// by filtering the event.
func (r handler) OnUpdate(oldObj, newObj interface{}) {
	r.filterEvent(newObj, "update")
}

// OnDelete adds a gvknn to the work queue
// by filtering the event.
func (r handler) OnDelete(obj interface{}) {
	r.filterEvent(obj, "delete")
}

// filterEvent adds a gvknn to the work queue when the object is
// - either found in the declared resources;
// - or managed by Config Sync.
func (r handler) filterEvent(obj interface{}, action string) {
	gvknn, err := gvknnFromObj(obj)
	if err != nil {
		glog.Warning(status.InternalErrorf("failed to get gvknn from object on %s %v", action, err))
		return
	}
	if _, found := r.resources.GetDecl(gvknn.ID); found {
		r.queue.Add(gvknn)
		return
	}

	// Pull metav1.Object out of the object.
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		glog.Warning(status.InternalErrorf("failed to get meta from object on %s %v", action, err))
		return
	}
	if managementEnabled(objMeta) {
		r.queue.Add(gvknn)
	}
}

// gvknnFromObj gets the GVKNN from an object
func gvknnFromObj(obj interface{}) (GVKNN, error) {
	// Pull the runtime.Object out of the object
	o, ok := obj.(runtime.Object)
	if !ok {
		return GVKNN{}, fmt.Errorf("missing runtime.Object")
	}
	gvk := o.GetObjectKind().GroupVersionKind()
	id, err := core.IDOfRuntime(o)
	if err != nil {
		return GVKNN{}, err
	}
	return GVKNN{
		ID:      id,
		Version: gvk.Version,
	}, nil
}

// enableManaged returns true if the resource explicitly has management enabled on a resource
// on the API server.
func managementEnabled(obj core.LabeledAndAnnotated) bool {
	return obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementEnabled
}
