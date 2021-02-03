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

const (
	// AnnotationKeyV1A2 is the annotation that indicates the namespace hierarchy is
	// not managed by the Hierarchical Namespace Controller (http://bit.ly/k8s-hnc-design) but
	// someone else, "configmanagement.gke.io" in this case.
	AnnotationKeyV1A2 = "hnc.x-k8s.io/managed-by"

	// AnnotationKeyV1A1 is the annotation that was used in HNC v0.5 (as part of its v1alpha1
	// interface). It should be removed after the first release of HNC v0.6, likely ACM 1.5.2.
	//
	// TODO(b/171305869): remove
	AnnotationKeyV1A1 = "hnc.x-k8s.io/managedBy"

	// DepthSuffix is a label suffix for hierarchical namespace depth.
	// See definition at http://bit.ly/k8s-hnc-design#heading=h.1wg2oqxxn6ka.
	DepthSuffix = ".tree.hnc.x-k8s.io/depth"
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
		core.SetAnnotation(newObject, AnnotationKeyV1A2, v1.ManagedByValue)
		core.SetAnnotation(newObject, AnnotationKeyV1A1, v1.ManagedByValue)
	}
	return newObject
}

// addDepthLabels adds depth labels to namespaces from its relative path. For
// example, for "namespaces/foo/bar/namespace.yaml", it will add the following
// two depth labels:
// - "foo.tree.hnc.x-k8s.io/depth: 1"
// - "bar.tree.hnc.x-k8s.io/depth: 0"
func addDepthLabels(o *ast.NamespaceObject, r cmpath.Relative) {
	// Relative path for namespaces should start with the "namespaces" directory,
	// include at least one directory matching the name of the namespace, and end
	// with "namespace.yaml". If not, early exit.
	p := r.Split()
	if len(p) < 3 {
		return
	}

	// Add depth labels for all names in the path except the first "namespaces"
	// directory and the last "namespace.yaml".
	p = p[1 : len(p)-1]

	for i, ans := range p {
		l := ans + DepthSuffix
		dist := strconv.Itoa(len(p) - i - 1)
		core.SetLabel(o.Object, l, dist)
	}
}
