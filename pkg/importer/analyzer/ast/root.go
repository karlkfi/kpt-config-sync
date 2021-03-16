package ast

// Root represents a hierarchy of declared configs, settings for how those configs will be
// interpreted, and information regarding where those configs came from.
type Root struct {
	// ClusterObjects represents resources that are cluster scoped.
	ClusterObjects []FileObject

	// ClusterRegistryObjects represents resources that are related to multi-cluster.
	ClusterRegistryObjects []FileObject

	// SystemObjects represents resources regarding nomos configuration.
	SystemObjects []FileObject

	// Tree represents the directory hierarchy containing namespace scoped resources.
	Tree *TreeNode
}
