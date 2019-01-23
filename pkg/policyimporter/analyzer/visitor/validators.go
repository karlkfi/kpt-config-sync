// Common ready-to-use validators which only require a validation function.

package visitor

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

type rootValidator struct {
	ValidatorBase
	validate func(g *ast.Root) error
}

// ValidateRoot implements Validator.
func (v *rootValidator) ValidateRoot(r *ast.Root) error {
	return v.validate(r)
}

// NewRootValidator returns a ValidatorVisitor which validates a Root with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewRootValidator(validate func(g *ast.Root) error) *ValidatorVisitor {
	return NewValidator(&rootValidator{validate: validate})
}

// NewTreeNodesValidator returns a ValidatorVisitor which validates all TreeNodes with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewTreeNodesValidator(validate func(ns []*ast.TreeNode) error) *ValidatorVisitor {
	var nodes []*ast.TreeNode
	validateRoot := func(g *ast.Root) error {
		return validate(nodes)
	}
	return NewRootValidator(validateRoot).WithPrerequisites(newNodeCollector(&nodes))
}

// systemValidator validates System.
type systemValidator struct {
	ValidatorBase
	validate func(s *ast.System) error
}

// ValidateSystem implements Validator.
func (v *systemValidator) ValidateSystem(s *ast.System) error {
	return v.validate(s)
}

// NewSystemValidator returns a ValidatorVisitor which validates System with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewSystemValidator(validate func(s *ast.System) error) *ValidatorVisitor {
	return NewValidator(&systemValidator{validate: validate})
}

type systemObjectValidator struct {
	ValidatorBase
	validate func(o *ast.SystemObject) error
}

// ValidateSystemObject implements Validator.
func (v *systemObjectValidator) ValidateSystemObject(o *ast.SystemObject) error {
	return v.validate(o)
}

// NewSystemObjectValidator returns a ValidatorVisitor which validates each SystemObject with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewSystemObjectValidator(validate func(o *ast.SystemObject) error) *ValidatorVisitor {
	return NewValidator(&systemObjectValidator{validate: validate})
}

type clusterRegistryValidator struct {
	ValidatorBase
	validate func(s *ast.ClusterRegistry) error
}

// ValidateClusterRegistry implements Validator.
func (v *clusterRegistryValidator) ValidateClusterRegistry(c *ast.ClusterRegistry) error {
	return v.validate(c)
}

// NewClusterRegistryValidator returns a ValidatorVisitor which validates ClusterRegistry with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewClusterRegistryValidator(validate func(s *ast.ClusterRegistry) error) *ValidatorVisitor {
	return NewValidator(&clusterRegistryValidator{validate: validate})
}

type clusterRegistryObjectValidator struct {
	ValidatorBase
	validate func(o *ast.ClusterRegistryObject) error
}

// ValidateClusterRegistryObject implements Validator.
func (v *clusterRegistryObjectValidator) ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) error {
	return v.validate(o)
}

// NewClusterRegistryObjectValidator returns a ValidatorVisitor which validates each ClusterRegistryObject with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewClusterRegistryObjectValidator(validate func(o *ast.ClusterRegistryObject) error) *ValidatorVisitor {
	return NewValidator(&clusterRegistryObjectValidator{validate: validate})
}

type clusterValidator struct {
	ValidatorBase
	validate func(s *ast.Cluster) error
}

// ValidateCluster implements Validator.
func (v *clusterValidator) ValidateCluster(c *ast.Cluster) error {
	return v.validate(c)
}

// NewClusterValidator returns a ValidatorVisitor which validates Cluster with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewClusterValidator(validate func(c *ast.Cluster) error) *ValidatorVisitor {
	return NewValidator(&clusterValidator{validate: validate})
}

type clusterObjectValidator struct {
	ValidatorBase
	validate func(o *ast.ClusterObject) error
}

// ValidateClusterObject implements Validator.
func (v *clusterObjectValidator) ValidateClusterObject(o *ast.ClusterObject) error {
	return v.validate(o)
}

// NewClusterObjectValidator returns a ValidatorVisitor which validates each ClusterObject with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewClusterObjectValidator(validate func(o *ast.ClusterObject) error) *ValidatorVisitor {
	return NewValidator(&clusterObjectValidator{validate: validate})
}

type treeNodeValidator struct {
	ValidatorBase
	validate func(n *ast.TreeNode) error
}

// ValidateTreeNode implements Validator.
func (v *treeNodeValidator) ValidateTreeNode(n *ast.TreeNode) error {
	return v.validate(n)
}

// NewTreeNodeValidator returns a ValidatorVisitor which validates each TreeNode with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewTreeNodeValidator(validate func(n *ast.TreeNode) error) *ValidatorVisitor {
	return NewValidator(&treeNodeValidator{validate: validate})
}

type objectValidator struct {
	ValidatorBase
	validate func(o *ast.NamespaceObject) error
}

// ValidateObject implements Validator.
func (v *objectValidator) ValidateObject(o *ast.NamespaceObject) error {
	return v.validate(o)
}

// NewObjectValidator returns a ValidatorVisitor which validates each NamespaceObject with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewObjectValidator(validate func(o *ast.NamespaceObject) error) *ValidatorVisitor {
	return NewValidator(&objectValidator{validate: validate})
}
