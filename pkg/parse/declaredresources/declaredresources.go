package declaredresources

import (
	"sync"

	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/nomos/pkg/core"
)

// DeclaredResources is the interface for providing resources from the filesystem to the remediator
type DeclaredResources struct {
	mutex     sync.RWMutex
	objectSet map[core.ID]*unstructured.Unstructured
}

// NewDeclaredResources creates an instance of DeclaredResources
func NewDeclaredResources() *DeclaredResources {
	return &DeclaredResources{
		mutex: sync.RWMutex{},
	}
}

// UpdateDecls performs an atomic update on the resource declaration set
func (dr *DeclaredResources) UpdateDecls(objects []core.Object) error {
	newSet := make(map[core.ID]*unstructured.Unstructured)
	for _, obj := range objects {
		id := core.IDOf(obj)
		u, err := reconcile.AsUnstructured(obj)
		if err != nil {
			// This should never happen.
			return errors.Wrapf(err, "converting %v to unstructured.Unstructured", id)
		}
		newSet[id] = u
	}
	dr.mutex.Lock()
	dr.objectSet = newSet
	dr.mutex.Unlock()
	return nil
}

// GetDecl returns the resource declaration as read from Git
func (dr *DeclaredResources) GetDecl(id core.ID) (*unstructured.Unstructured, bool) {
	dr.mutex.RLock()
	u, found := dr.objectSet[id]
	dr.mutex.RUnlock()
	return u, found
}

// Decls returns all declarations from Git.
func (dr *DeclaredResources) Decls() []*unstructured.Unstructured {
	var objects []*unstructured.Unstructured
	dr.mutex.RLock()
	objSet := dr.objectSet
	dr.mutex.RUnlock()
	for _, obj := range objSet {
		objects = append(objects, obj)
	}
	return objects
}

// GetGKSet returns the set of all GroupKind found in the git repo.
func (dr *DeclaredResources) GetGKSet() map[schema.GroupKind]string {
	gkSet := make(map[schema.GroupKind]string)
	dr.mutex.RLock()
	objSet := dr.objectSet
	dr.mutex.RUnlock()
	for id, obj := range objSet {
		gk := id.GroupKind
		if _, ok := gkSet[gk]; !ok {
			gkSet[gk] = obj.GroupVersionKind().Version
		}
	}
	return gkSet
}
