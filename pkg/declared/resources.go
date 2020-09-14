package declared

import (
	"sync"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/nomos/pkg/core"
)

// Resources is a threadsafe container for a set of resources declared in a Git
// repo.
type Resources struct {
	mutex sync.RWMutex
	// objectSet is a map of object IDs to the unstructured format of those
	// objects. Note that the pointer to this map is threadsafe but the map itself
	// is not threadsafe. This map should never be returned from a function
	// directly. The map should never be written to once it has been assigned to
	// this reference; it should be treated as read-only from then on.
	objectSet map[core.ID]*unstructured.Unstructured
}

// Update performs an atomic update on the resource declaration set.
func (r *Resources) Update(objects []core.Object) status.Error {
	// First build up the new map using a local pointer/reference.
	newSet := make(map[core.ID]*unstructured.Unstructured)
	for _, obj := range objects {
		if obj == nil {
			glog.Warning("Resources received nil declared resource")
			// TODO(b/162601559): Increment internal error metric here
			continue
		}
		id := core.IDOf(obj)
		u, err := reconcile.AsUnstructuredSanitized(obj)
		if err != nil {
			// This should never happen.
			return status.InternalErrorBuilder.Wrap(err).
				Sprintf("converting %v to unstructured.Unstructured", id).Build()
		}
		newSet[id] = u
	}
	// Now assign the pointer for the new map to the struct reference in a
	// threadsafe context. From now on, this map is read-only.
	r.mutex.Lock()
	r.objectSet = newSet
	r.mutex.Unlock()
	return nil
}

// Get returns a copy of the resource declaration as read from Git
func (r *Resources) Get(id core.ID) (*unstructured.Unstructured, bool) {
	r.mutex.RLock()
	objSet := r.objectSet
	r.mutex.RUnlock()

	// A local reference to the map is threadsafe since only the struct reference
	// is replaced on update.
	u, found := objSet[id]
	// We return a copy of the Unstructured, as
	// 1) client.Client methods mutate the objects passed into them.
	// 2) We don't want to persist any changes made to an object we retrieved
	//  from a declared.Resources.
	return u.DeepCopy(), found
}

// Declarations returns all resource declarations from Git.
func (r *Resources) Declarations() []*unstructured.Unstructured {
	var objects []*unstructured.Unstructured
	r.mutex.RLock()
	objSet := r.objectSet
	r.mutex.RUnlock()

	// A local reference to the map is threadsafe since only the struct reference
	// is replaced on update.
	for _, obj := range objSet {
		objects = append(objects, obj)
	}
	return objects
}

// GVKSet returns the set of all GroupVersionKind found in the git repo.
func (r *Resources) GVKSet() map[schema.GroupVersionKind]bool {
	gvkSet := make(map[schema.GroupVersionKind]bool)
	r.mutex.RLock()
	objSet := r.objectSet
	r.mutex.RUnlock()

	// A local reference to the objSet map is threadsafe since only the pointer to
	// the map is replaced on update.
	for _, obj := range objSet {
		gvk := obj.GroupVersionKind()
		if !gvkSet[gvk] {
			gvkSet[gvk] = true
		}
	}
	return gvkSet
}
