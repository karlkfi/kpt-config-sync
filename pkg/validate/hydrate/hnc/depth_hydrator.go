package hnc

import (
	"strconv"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	oldhnc "github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
)

type depthHydrator struct{}

var _ parsed.TreeHydrator = &depthHydrator{}

// DepthHydrator returns a TreeHydrator that annotates each Namespace with its
// depth to be compatible with the Hierarchy Namespace Controller.
func DepthHydrator() parsed.TreeHydrator {
	return &depthHydrator{}
}

// Hydrate implements TreeHydrator.
func (d *depthHydrator) Hydrate(root *parsed.TreeRoot) status.MultiError {
	return root.VisitNamespaceObjects(parsed.PerObjectVisitor(addDepthMetadata))
}

func addDepthMetadata(obj ast.FileObject) status.Error {
	if obj.GroupVersionKind() == kinds.Namespace() {
		addDepthLabels(obj)
		core.SetAnnotation(obj, oldhnc.AnnotationKeyV1A2, v1.ManagedByValue)
		core.SetAnnotation(obj, oldhnc.AnnotationKeyV1A1, v1.ManagedByValue)
	}
	return nil
}

// addDepthLabels adds depth labels to namespaces from its relative path. For
// example, for "namespaces/foo/bar/namespace.yaml", it will add the following
// two depth labels:
// - "foo.tree.hnc.x-k8s.io/depth: 1"
// - "bar.tree.hnc.x-k8s.io/depth: 0"
func addDepthLabels(obj ast.FileObject) {
	// Relative path for namespaces should start with the "namespaces" directory,
	// include at least one directory matching the name of the namespace, and end
	// with "namespace.yaml". If not, early exit.
	p := obj.Split()
	if len(p) < 3 {
		return
	}

	// Add depth labels for all names in the path except the first "namespaces"
	// directory and the last "namespace.yaml".
	p = p[1 : len(p)-1]

	for i, ans := range p {
		l := ans + oldhnc.DepthSuffix
		dist := strconv.Itoa(len(p) - i - 1)
		core.SetLabel(obj, l, dist)
	}
}
