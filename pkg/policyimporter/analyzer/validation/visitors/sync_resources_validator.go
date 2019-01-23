package visitors

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewSyncResourcesValidator initializes a ValidatorVisitor that ensures all NamespaceObjects have syncs.
func NewSyncResourcesValidator() *visitor.ValidatorVisitor {
	syncs := make(map[schema.GroupVersionKind]bool)

	ensureSyncd := func(o *ast.NamespaceObject) error {
		if !syncs[o.GroupVersionKind()] {
			return vet.UnsyncableNamespaceObjectError{Resource: o}
		}
		return nil
	}

	// Passes the map of available syncs to the syncGVKCollector so ensureSyncd has access to it.
	return visitor.NewObjectValidator(ensureSyncd).WithPrerequisites(NewSyncGVKCollector(syncs))
}

// syncGVKCollector collects all Sync declarations into a map of schema.GroupVersionKind.
type syncGVKCollector struct {
	*visitor.Base
	syncs map[schema.GroupVersionKind]bool
}

// NewSyncGVKCollector initializes a syncGVKCollector with an output map of syncs.
func NewSyncGVKCollector(syncs map[schema.GroupVersionKind]bool) ast.Visitor {
	v := &syncGVKCollector{Base: visitor.NewBase(), syncs: syncs}
	v.SetImpl(v)
	return v
}

// VisitSystemObject adds any Syncs declared in the object to the map.
func (sc *syncGVKCollector) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	switch s := o.Object.(type) {
	case *v1alpha1.Sync:
		for _, group := range s.Spec.Groups {
			for _, kind := range group.Kinds {
				for _, version := range kind.Versions {
					gvk := schema.GroupVersionKind{Group: group.Group, Version: version.Version, Kind: kind.Kind}
					sc.syncs[gvk] = true
				}
			}
		}
	}
	return o
}
