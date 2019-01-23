package visitor

import "github.com/google/nomos/pkg/policyimporter/analyzer/ast"

// Validator defines validation that happens at different levels of the policy hierarchy.
//
// Returning an error indicates a problem; returning nil indicates the repository has passed this
// Validator.
type Validator interface {
	// ValidateRoot defines validation that happens at the Root of the repository.
	ValidateRoot(r *ast.Root) error

	// ValidateSystem defines validation that happens on the system/ directory.
	ValidateSystem(s *ast.System) error

	// ValidateSystemObject defines validation that happens on each object in the system/ directory.
	ValidateSystemObject(o *ast.SystemObject) error

	// ValidateClusterRegistry defines validation that happens on the clusterregistry/ directory.
	ValidateClusterRegistry(c *ast.ClusterRegistry) error

	// ValidateClusterRegistryObject defines validation that happens on each object in the clusterregistry/ directory.
	ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) error

	// ValidateCluster defines validation that happens on the cluster/ directory.
	ValidateCluster(c *ast.Cluster) error

	// ValidateClusterObject defines validation that happens on each object in the cluster/ directory.
	ValidateClusterObject(o *ast.ClusterObject) error

	// ValidateTreeNode defines validation that happens on each node in the policy hierarchy.
	// For nomos, this is namespaces/.
	// For bespin, this is hierarchy/.
	ValidateTreeNode(n *ast.TreeNode) error

	// ValidateObject defines validation that happens on each object in the policy hierarchy.
	ValidateObject(o *ast.NamespaceObject) error
}
