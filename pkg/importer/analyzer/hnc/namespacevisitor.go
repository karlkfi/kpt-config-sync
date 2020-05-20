// Package hnc adds additional HNC-understandable annotation and labels to namespaces managed by
// ACM. Please send code reviews to gke-kubernetes-hnc-core@.
package hnc

import (
	"strconv"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
)

// namespaceVisitor sets hierarchy controller annotation and labels on namespaces.
type namespaceVisitor struct {
	*visitor.Base
}

var _ ast.Visitor = &namespaceVisitor{}

// NewNamespaceVisitor returns a new namespaceVisitor
func NewNamespaceVisitor() ast.Visitor {
	v := &namespaceVisitor{
		Base: visitor.NewBase(),
	}
	v.SetImpl(v)
	return v
}

// VisitObject implements Visitor
func (v *namespaceVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	newObject := v.Base.VisitObject(o)
	if newObject.GroupVersionKind() == kinds.Namespace() {
		addDepthLabels(newObject, newObject.Relative)
		core.SetAnnotation(newObject, v1.HierarchyControllerAnnotationKey, v1.ManagedByValue)
	}
	return newObject
}

// addDepthLabels adds depth labels to namespaces from its relative path. For example,
// for "namespaces/dev/namespace.yaml", it will add the following two depth labels:
// - "dev.tree.hnc.x-k8s.io/depth: 0"
// - "config-sync-root.tree.hnc.x-k8s.io/depth: 1"
// Note: "config-sync-root" is used as root in the hierarchy.
func addDepthLabels(o *ast.NamespaceObject, r cmpath.Relative) {
	// Relative path for namespaces should start with "namespaces" and end with "namespace.yaml".
	// If not, early exit.
	p := r.Split()
	if len(p) < 2 {
		return
	}

	// Replace "namespaces" with "config-sync-root" as the root in the hierarchy and
	// add depth labels for all names in the path except the last "namespace.yaml".
	p[0] = v1.DepthLabelRootName
	p = p[:len(p)-1]
	for i, ans := range p {
		l := ans + v1.HierarchyControllerDepthSuffix
		dist := strconv.Itoa(len(p) - i - 1)
		core.SetLabel(o.Object, l, dist)
	}
}
