package filesystem

import "github.com/google/nomos/pkg/importer/analyzer/ast"

// FlatRoot is a collection of objects by major directory.
type FlatRoot struct {
	SystemObjects          []ast.FileObject
	ClusterRegistryObjects []ast.FileObject
	ClusterObjects         []ast.FileObject
	NamespaceObjects       []ast.FileObject
}
