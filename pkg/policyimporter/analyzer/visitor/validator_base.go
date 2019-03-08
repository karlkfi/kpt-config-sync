package visitor

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// ValidatorBase provides a default implementation of Validator which does nothing.
type ValidatorBase struct{}

var _ Validator = &ValidatorBase{}

// ValidateRoot implements Validator.
func (vb *ValidatorBase) ValidateRoot(g *ast.Root) *status.MultiError {
	return nil
}

// ValidateSystem implements Validator.
func (vb *ValidatorBase) ValidateSystem(c []*ast.SystemObject) *status.MultiError {
	return nil
}

// ValidateSystemObject implements Validator.
func (vb *ValidatorBase) ValidateSystemObject(o *ast.SystemObject) *status.MultiError {
	return nil
}

// ValidateClusterRegistry implements Validator.
func (vb *ValidatorBase) ValidateClusterRegistry(c []*ast.ClusterRegistryObject) *status.MultiError {
	return nil
}

// ValidateClusterRegistryObject implements Validator.
func (vb *ValidatorBase) ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) *status.MultiError {
	return nil
}

// ValidateCluster implements Validator.
func (vb *ValidatorBase) ValidateCluster(c []*ast.ClusterObject) *status.MultiError {
	return nil
}

// ValidateClusterObject implements Validator.
func (vb *ValidatorBase) ValidateClusterObject(o *ast.ClusterObject) *status.MultiError {
	return nil
}

// ValidateTreeNode implements Validator.
func (vb *ValidatorBase) ValidateTreeNode(n *ast.TreeNode) *status.MultiError {
	return nil
}

// ValidateObject implements Validator.
func (vb *ValidatorBase) ValidateObject(o *ast.NamespaceObject) *status.MultiError {
	return nil
}
