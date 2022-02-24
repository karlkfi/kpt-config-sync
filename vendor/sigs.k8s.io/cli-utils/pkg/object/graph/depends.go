// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

// This package provides a object sorting functionality
// based on the explicit "depends-on" annotation, and
// implicit object dependencies like namespaces and CRD's.
package graph

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/multierror"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/object/dependson"
	"sigs.k8s.io/cli-utils/pkg/object/mutation"
	"sigs.k8s.io/cli-utils/pkg/object/validation"
	"sigs.k8s.io/cli-utils/pkg/ordering"
)

// SortObjs returns a slice of the sets of objects to apply (in order).
// Each of the objects in an apply set is applied together. The order of
// the returned applied sets is a topological ordering of the sets to apply.
// Returns an single empty apply set if there are no objects to apply.
func SortObjs(objs object.UnstructuredSet) ([]object.UnstructuredSet, error) {
	var objSets []object.UnstructuredSet
	if len(objs) == 0 {
		return objSets, nil
	}
	var errors []error
	// Convert to IDs (same length & order as objs)
	ids := object.UnstructuredSetToObjMetadataSet(objs)
	// Create the graph, and build a map of object metadata to the object (Unstructured).
	g := New()
	objToUnstructured := map[object.ObjMetadata]*unstructured.Unstructured{}
	for i, obj := range objs {
		id := ids[i]
		objToUnstructured[id] = obj
	}
	// Add objects as graph vertices
	addVertices(g, ids)
	// Add dependencies as graph edges
	addCRDEdges(g, objs, ids)
	addNamespaceEdges(g, objs, ids)
	if err := addDependsOnEdges(g, objs, ids); err != nil {
		errors = append(errors, err)
	}
	if err := addApplyTimeMutationEdges(g, objs, ids); err != nil {
		errors = append(errors, err)
	}
	// Run topological sort on the graph.
	sortedObjSets, err := g.Sort()
	if err != nil {
		errors = append(errors, err)
	}
	// Map the object metadata back to the sorted sets of unstructured objects.
	// Ignore any edges that aren't part of the input set (don't wait for them).
	for _, objSet := range sortedObjSets {
		currentSet := object.UnstructuredSet{}
		for _, id := range objSet {
			var found bool
			var obj *unstructured.Unstructured
			if obj, found = objToUnstructured[id]; found {
				currentSet = append(currentSet, obj)
			}
		}
		// Sort each set in apply order
		sort.Sort(ordering.SortableUnstructureds(currentSet))
		objSets = append(objSets, currentSet)
	}
	if len(errors) > 0 {
		return objSets, multierror.Wrap(errors...)
	}
	return objSets, nil
}

// ReverseSortObjs is the same as SortObjs but using reverse ordering.
func ReverseSortObjs(objs object.UnstructuredSet) ([]object.UnstructuredSet, error) {
	// Sorted objects using normal ordering.
	s, err := SortObjs(objs)
	if err != nil {
		return s, err
	}
	// Reverse the ordering of the object sets using swaps.
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	// Reverse the ordering of the objects in each set using swaps.
	for _, c := range s {
		for i, j := 0, len(c)-1; i < j; i, j = i+1, j-1 {
			c[i], c[j] = c[j], c[i]
		}
	}
	return s, nil
}

// addVertices adds all the IDs in the set as graph vertices.
func addVertices(g *Graph, ids object.ObjMetadataSet) {
	for _, id := range ids {
		klog.V(3).Infof("adding vertex: %s", id)
		g.AddVertex(id)
	}
}

// addApplyTimeMutationEdges updates the graph with edges from objects
// with an explicit "apply-time-mutation" annotation.
// The objs and ids must match in order and length (optimization).
func addApplyTimeMutationEdges(g *Graph, objs object.UnstructuredSet, ids object.ObjMetadataSet) error {
	var errors []error
	for i, obj := range objs {
		if !mutation.HasAnnotation(obj) {
			continue
		}
		id := ids[i]
		subs, err := mutation.ReadAnnotation(obj)
		if err != nil {
			klog.V(3).Infof("failed to add edges from: %s: %v", id, err)
			errors = append(errors, validation.NewError(err, id))
			continue
		}
		seen := make(map[object.ObjMetadata]struct{})
		var objErrors []error
		for _, sub := range subs {
			dep := sub.SourceRef.ToObjMetadata()
			// Duplicate dependencies can be safely skipped.
			if _, found := seen[dep]; found {
				continue
			}
			// Mark as seen
			seen[dep] = struct{}{}
			// Require dependencies to be in the same resource group.
			// Waiting for external dependencies isn't implemented (yet).
			if !ids.Contains(dep) {
				err := object.InvalidAnnotationError{
					Annotation: mutation.Annotation,
					Cause: ExternalDependencyError{
						Edge: Edge{
							From: id,
							To:   dep,
						},
					},
				}
				objErrors = append(objErrors, err)
				klog.V(3).Infof("failed to add edges: %v", err)
				continue
			}
			klog.V(3).Infof("adding edge from: %s, to: %s", id, dep)
			g.AddEdge(id, dep)
		}
		if len(objErrors) > 0 {
			errors = append(errors,
				validation.NewError(multierror.Wrap(objErrors...), id))
		}
	}
	if len(errors) > 0 {
		return multierror.Wrap(errors...)
	}
	return nil
}

// addDependsOnEdges updates the graph with edges from objects
// with an explicit "depends-on" annotation.
// The objs and ids must match in order and length (optimization).
func addDependsOnEdges(g *Graph, objs object.UnstructuredSet, ids object.ObjMetadataSet) error {
	var errors []error
	for i, obj := range objs {
		if !dependson.HasAnnotation(obj) {
			continue
		}
		id := ids[i]
		deps, err := dependson.ReadAnnotation(obj)
		if err != nil {
			klog.V(3).Infof("failed to add edges from: %s: %v", id, err)
			errors = append(errors, validation.NewError(err, id))
			continue
		}
		seen := make(map[object.ObjMetadata]struct{})
		var objErrors []error
		for _, dep := range deps {
			// Duplicate dependencies in the same annotation are not allowed.
			// Having duplicates won't break the graph, but skip it anyway.
			if _, found := seen[dep]; found {
				err := object.InvalidAnnotationError{
					Annotation: dependson.Annotation,
					Cause: DuplicateDependencyError{
						Edge: Edge{
							From: id,
							To:   dep,
						},
					},
				}
				objErrors = append(objErrors, err)
				klog.V(3).Infof("failed to add edges from: %s: %v", id, err)
				continue
			}
			// Mark as seen
			seen[dep] = struct{}{}
			// Require dependencies to be in the same resource group.
			// Waiting for external dependencies isn't implemented (yet).
			if !ids.Contains(dep) {
				err := object.InvalidAnnotationError{
					Annotation: dependson.Annotation,
					Cause: ExternalDependencyError{
						Edge: Edge{
							From: id,
							To:   dep,
						},
					},
				}
				objErrors = append(objErrors, err)
				klog.V(3).Infof("failed to add edges: %v", err)
				continue
			}
			klog.V(3).Infof("adding edge from: %s, to: %s", id, dep)
			g.AddEdge(id, dep)
		}
		if len(objErrors) > 0 {
			errors = append(errors,
				validation.NewError(multierror.Wrap(objErrors...), id))
		}
	}
	if len(errors) > 0 {
		return multierror.Wrap(errors...)
	}
	return nil
}

// addCRDEdges adds edges to the dependency graph from custom
// resources to their definitions to ensure the CRD's exist
// before applying the custom resources created with the definition.
// The objs and ids must match in order and length (optimization).
func addCRDEdges(g *Graph, objs object.UnstructuredSet, ids object.ObjMetadataSet) {
	crds := map[string]object.ObjMetadata{}
	// First create a map of all the CRD's.
	for i, u := range objs {
		if object.IsCRD(u) {
			groupKind, found := object.GetCRDGroupKind(u)
			if found {
				crds[groupKind.String()] = ids[i]
			}
		}
	}
	// Iterate through all resources to see if we are applying any
	// custom resources defined by previously recorded CRD's.
	for i, u := range objs {
		gvk := u.GroupVersionKind()
		groupKind := gvk.GroupKind()
		if to, found := crds[groupKind.String()]; found {
			from := ids[i]
			klog.V(3).Infof("adding edge from: custom resource %s, to CRD: %s", from, to)
			g.AddEdge(from, to)
		}
	}
}

// addNamespaceEdges adds edges to the dependency graph from namespaced
// objects to the namespace objects. Ensures the namespaces exist
// before the resources in those namespaces are applied.
// The objs and ids must match in order and length (optimization).
func addNamespaceEdges(g *Graph, objs object.UnstructuredSet, ids object.ObjMetadataSet) {
	namespaces := map[string]object.ObjMetadata{}
	// First create a map of all the namespaces objects live in.
	for i, obj := range objs {
		if object.IsKindNamespace(obj) {
			namespace := obj.GetName()
			namespaces[namespace] = ids[i]
		}
	}
	// Next, if the namespace of a namespaced object is being applied,
	// then create an edge from the namespaced object to its namespace.
	for i, obj := range objs {
		if object.IsNamespaced(obj) {
			objNamespace := obj.GetNamespace()
			if to, found := namespaces[objNamespace]; found {
				from := ids[i]
				klog.V(3).Infof("adding edge from: %s to namespace: %s", from, to)
				g.AddEdge(from, to)
			}
		}
	}
}
