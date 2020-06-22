package declaredresources

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
)

// DeclaredResources is the interface for providing resources from the filesystem to the remediator
type DeclaredResources struct {
	mutex     sync.RWMutex
	objectSet map[core.ID]*ast.FileObject
}

// NewDeclaredResources creates an instance of DeclaredResources
func NewDeclaredResources() *DeclaredResources {
	return &DeclaredResources{
		mutex: sync.RWMutex{},
	}
}

// UpdateDecls performs an atomic update on the resource declaration set
func (dr *DeclaredResources) UpdateDecls(objects []ast.FileObject) error {
	newSet := make(map[core.ID]*ast.FileObject)
	for _, obj := range objects {
		id := core.IDOf(obj)
		newSet[id] = &obj
	}
	dr.mutex.Lock()
	dr.objectSet = newSet
	dr.mutex.Unlock()
	return nil
}

// GetDecl returns the resource declaration as read from Git
func (dr *DeclaredResources) GetDecl(obj runtime.Object) (*ast.FileObject, error) {
	o, err := core.ObjectOf(obj)
	if err != nil {
		return nil, err
	}
	id := core.IDOf(o)
	dr.mutex.RLock()
	fileObj, ok := dr.objectSet[id]
	dr.mutex.RUnlock()
	if ok {
		return fileObj, nil
	}
	return nil, fmt.Errorf("id:%v not found in declared resources", id)
}

// Decls returns all declarations from Git.
func (dr *DeclaredResources) Decls() []*ast.FileObject {
	var objects []*ast.FileObject
	dr.mutex.RLock()
	objSet := dr.objectSet
	dr.mutex.RUnlock()
	for _, obj := range objSet {
		objects = append(objects, obj)
	}
	return objects
}

// GetGKSet returns the set of all GroupKind found in the git repo.
func (dr *DeclaredResources) GetGKSet() map[schema.GroupKind]struct{} {
	gkSet := make(map[schema.GroupKind]struct{})
	dr.mutex.RLock()
	objSet := dr.objectSet
	dr.mutex.RUnlock()
	for id := range objSet {
		gk := id.GroupKind
		if _, ok := gkSet[gk]; !ok {
			gkSet[gk] = struct{}{}
		}
	}
	return gkSet
}
