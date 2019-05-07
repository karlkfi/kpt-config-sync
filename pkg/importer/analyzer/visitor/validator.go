package visitor

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// Validator defines validation that happens at different levels of the config hierarchy.
//
// Returning an error indicates a problem; returning nil indicates the repository has passed this
// Validator.
type Validator interface {
	// ValidateRoot defines validation that happens at the Root of the repository.
	ValidateRoot(r *ast.Root) status.MultiError

	// ValidateSystem defines validation that happens on the system/ directory.
	ValidateSystem(s []*ast.SystemObject) status.MultiError

	// ValidateSystemObject defines validation that happens on each object in the system/ directory.
	ValidateSystemObject(o *ast.SystemObject) status.MultiError

	// ValidateClusterRegistry defines validation that happens on the clusterregistry/ directory.
	ValidateClusterRegistry(c []*ast.ClusterRegistryObject) status.MultiError

	// ValidateClusterRegistryObject defines validation that happens on each object in the clusterregistry/ directory.
	ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) status.MultiError

	// ValidateCluster defines validation that happens on the cluster/ directory.
	ValidateCluster(c []*ast.ClusterObject) status.MultiError

	// ValidateClusterObject defines validation that happens on each object in the cluster/ directory.
	ValidateClusterObject(o *ast.ClusterObject) status.MultiError

	// ValidateTreeNode defines validation that happens on each node in the config hierarchy.
	// For nomos, this is namespaces/.
	// For bespin, this is hierarchy/.
	ValidateTreeNode(n *ast.TreeNode) status.MultiError

	// ValidateObject defines validation that happens on each object in the config hierarchy.
	ValidateObject(o *ast.NamespaceObject) status.MultiError
}
