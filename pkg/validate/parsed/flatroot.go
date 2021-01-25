package parsed

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// FlatRoot is an unstructured collection of declared configs.
type FlatRoot struct {
	// ClusterObjects represents resources that are cluster scoped.
	ClusterObjects []ast.FileObject
	// ClusterRegistryObjects represents resources that are related to multi-cluster.
	ClusterRegistryObjects []ast.FileObject
	// NamespaceObjects represents resources that are namespace-scoped.
	NamespaceObjects []ast.FileObject
	// SystemObjects represents resources regarding ConfigSync configuration.
	SystemObjects []ast.FileObject
}

var _ Root = &FlatRoot{}

// VisitAllObjects implements Root.
func (f *FlatRoot) VisitAllObjects(visit VisitorFunc) status.MultiError {
	err := f.VisitSystemObjects(visit)
	err = status.Append(err, f.VisitClusterRegistryObjects(visit))
	err = status.Append(err, f.VisitClusterObjects(visit))
	return status.Append(err, f.VisitNamespaceObjects(visit))
}

// VisitClusterObjects implements Root.
func (f *FlatRoot) VisitClusterObjects(visit VisitorFunc) status.MultiError {
	return visit(f.ClusterObjects)
}

// VisitClusterRegistryObjects implements Root.
func (f *FlatRoot) VisitClusterRegistryObjects(visit VisitorFunc) status.MultiError {
	return visit(f.ClusterRegistryObjects)
}

// VisitNamespaceObjects implements Root.
func (f *FlatRoot) VisitNamespaceObjects(visit VisitorFunc) status.MultiError {
	return visit(f.NamespaceObjects)
}

// VisitSystemObjects implements Root.
func (f *FlatRoot) VisitSystemObjects(visit VisitorFunc) status.MultiError {
	return visit(f.SystemObjects)
}
