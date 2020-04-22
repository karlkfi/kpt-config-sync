// Common ready-to-use validators which only require a validation function.

package visitor

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

type rootValidator struct {
	ValidatorBase
	validate func(g *ast.Root) status.MultiError
}

// ValidateRoot implements Validator.
func (v *rootValidator) ValidateRoot(r *ast.Root) status.MultiError {
	return v.validate(r)
}

// NewRootValidator returns a ValidatorVisitor which validates a Root with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewRootValidator(validate func(g *ast.Root) status.MultiError) *ValidatorVisitor {
	return NewValidator(&rootValidator{validate: validate})
}

type systemObjectValidator struct {
	ValidatorBase
	validate func(o *ast.SystemObject) status.MultiError
}

// ValidateSystemObject implements Validator.
func (v *systemObjectValidator) ValidateSystemObject(o *ast.SystemObject) status.MultiError {
	return v.validate(o)
}

// NewSystemObjectValidator returns a ValidatorVisitor which validates each SystemObject with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewSystemObjectValidator(validate func(o *ast.SystemObject) status.MultiError) *ValidatorVisitor {
	return NewValidator(&systemObjectValidator{validate: validate})
}

type clusterObjectValidator struct {
	ValidatorBase
	validate func(o *ast.ClusterObject) status.MultiError
}

// ValidateClusterObject implements Validator.
func (v *clusterObjectValidator) ValidateClusterObject(o *ast.ClusterObject) status.MultiError {
	return v.validate(o)
}

// NewClusterObjectValidator returns a ValidatorVisitor which validates each ClusterObject with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewClusterObjectValidator(validate func(o *ast.ClusterObject) status.MultiError) *ValidatorVisitor {
	return NewValidator(&clusterObjectValidator{validate: validate})
}

type treeNodeValidator struct {
	ValidatorBase
	validate func(n *ast.TreeNode) status.MultiError
}

// ValidateTreeNode implements Validator.
func (v *treeNodeValidator) ValidateTreeNode(n *ast.TreeNode) status.MultiError {
	return v.validate(n)
}

// NewTreeNodeValidator returns a ValidatorVisitor which validates each TreeNode with the passed validation function.
// Errors returned by validate during visiting will be returned by Error().
func NewTreeNodeValidator(validate func(n *ast.TreeNode) status.MultiError) *ValidatorVisitor {
	return NewValidator(&treeNodeValidator{validate: validate})
}

type allObjectValidator struct {
	ValidatorBase
	validate func(o ast.FileObject) status.MultiError
}

// NewAllObjectValidator returns a ValidatorVisitor which validates every Resource's metadata fields.
// Validates every SystemObject, ClusterRegistryObject, ClusterObject, and NamespaceObject.
func NewAllObjectValidator(validate func(o ast.FileObject) status.MultiError) *ValidatorVisitor {
	return NewValidator(&allObjectValidator{validate: validate})
}

// ValidateSystemObject implements Validator.
func (v *allObjectValidator) ValidateSystemObject(o *ast.SystemObject) status.MultiError {
	return v.validate(o.FileObject)
}

// ValidateClusterRegistryObject implements Validator.
func (v *allObjectValidator) ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) status.MultiError {
	return v.validate(o.FileObject)
}

// ValidateClusterObject implements Validator.
func (v *allObjectValidator) ValidateClusterObject(o *ast.ClusterObject) status.MultiError {
	return v.validate(o.FileObject)
}

// ValidateObject implements Validator.
func (v *allObjectValidator) ValidateObject(o *ast.NamespaceObject) status.MultiError {
	return v.validate(o.FileObject)
}

type allNodesValidator struct {
	ValidatorBase
	validate func(os []ast.FileObject) status.MultiError
}

// NewAllNodesValidator returns a validator which applies the same validation to every node,
// including the non-hierarchical configs. For now this is just the objects in each nodes, as
// that is the single unifying similarity.
func NewAllNodesValidator(validate func(os []ast.FileObject) status.MultiError) *ValidatorVisitor {
	return NewValidator(&allNodesValidator{validate: validate})
}

// ValidateSystem implements Validator.
func (v *allNodesValidator) ValidateSystem(o []*ast.SystemObject) status.MultiError {
	objects := make([]ast.FileObject, len(o))
	for i, o := range o {
		objects[i] = o.FileObject
	}
	return v.validate(objects)
}

// ValidateClusterRegistry implements Validator.
func (v *allNodesValidator) ValidateClusterRegistry(o []*ast.ClusterRegistryObject) status.MultiError {
	objects := make([]ast.FileObject, len(o))
	for i, o := range o {
		objects[i] = o.FileObject
	}
	return v.validate(objects)
}

// ValidateCluster implements Validator.
func (v *allNodesValidator) ValidateCluster(o []*ast.ClusterObject) status.MultiError {
	objects := make([]ast.FileObject, len(o))
	for i, o := range o {
		objects[i] = o.FileObject
	}
	return v.validate(objects)
}

// ValidateTreeNode implements Validator.
func (v *allNodesValidator) ValidateTreeNode(o *ast.TreeNode) status.MultiError {
	objects := make([]ast.FileObject, len(o.Objects))
	for i, o := range o.Objects {
		objects[i] = o.FileObject
	}
	return v.validate(objects)
}
