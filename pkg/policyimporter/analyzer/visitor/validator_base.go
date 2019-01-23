package visitor

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// ValidatorBase provides a default implementation of Validator which does nothing.
type ValidatorBase struct{}

var _ Validator = &ValidatorBase{}

// ValidateRoot implements Validator.
func (vb *ValidatorBase) ValidateRoot(g *ast.Root) error {
	return nil
}

// ValidateSystem implements Validator.
func (vb *ValidatorBase) ValidateSystem(c *ast.System) error {
	return nil
}

// ValidateSystemObject implements Validator.
func (vb *ValidatorBase) ValidateSystemObject(o *ast.SystemObject) error {
	return nil
}

// ValidateClusterRegistry implements Validator.
func (vb *ValidatorBase) ValidateClusterRegistry(c *ast.ClusterRegistry) error {
	return nil
}

// ValidateClusterRegistryObject implements Validator.
func (vb *ValidatorBase) ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) error {
	return nil
}

// ValidateCluster implements Validator.
func (vb *ValidatorBase) ValidateCluster(c *ast.Cluster) error {
	return nil
}

// ValidateClusterObject implements Validator.
func (vb *ValidatorBase) ValidateClusterObject(o *ast.ClusterObject) error {
	return nil
}

// ValidateTreeNode implements Validator.
func (vb *ValidatorBase) ValidateTreeNode(n *ast.TreeNode) error {
	return nil
}

// ValidateObject implements Validator.
func (vb *ValidatorBase) ValidateObject(o *ast.NamespaceObject) error {
	return nil
}
