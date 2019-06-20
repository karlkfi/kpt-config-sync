package ast

// FlatRoot is a collection of objects by major directory.
type FlatRoot struct {
	SystemObjects          []FileObject
	ClusterRegistryObjects []FileObject
	ClusterObjects         []FileObject
	NamespaceObjects       []FileObject
}
