package visitor

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/util/multierror"
)

// ValidatorVisitor provides the basic necessary functionality for validators.
//
// Inherits traversal order from visitor.Base.
//
// Example 1: Ensure every Namespace object is named "correct name".
//
//    var MyExample1 = visitor.NewObjectValidator(
//      func(o *ast.NamespaceObject) error {
//        if o.Name() != "correct name" {
//          return vet.UndocumentedError("Incorrect name %q", o.Name())
//        }
//        return nil
//      }
//    )
//
// Example 2: Ensure no TreeNode declares multiple Roles.
//
//    var MyExample2 = visitor.NewTreeNodeValidator(
//      func(n *ast.TreeNode) error {
//        var roles []ast.NamespaceObject
//        for _, o := n.Objects() {
//          if o.GroupVersionKind() == kinds.Role() {
//            roles = append(roles, o)
//          }
//        }
//        if len(roles) > 1 {
//          return vet.UndocumentedError("Multiple roles defined in %q", n.RelativeSlashPath())
//        }
//        return nil
//      }
//    )
type ValidatorVisitor struct {
	*Base
	prerequisites []ast.Visitor
	validator     Validator
	errors        multierror.Builder
}

var _ ast.Visitor = &ValidatorVisitor{}

// NewValidator initializes a ValidatorVisitor.
// validator need not inherit from visitor.Base to function.
func NewValidator(underlying Validator) *ValidatorVisitor {
	v := &ValidatorVisitor{Base: NewBase(), validator: underlying}
	v.SetImpl(v)
	return v
}

// WithPrerequisites adds a prerequisite visitor which is run before any other validation.
//
// Prerequisites are run sequentially in the order they were added.
// Prerequisites do not modify the pointer to root.
// Prerequisites should inherit from visitor.Base to ensure all of their Visit methods are called.
func (v *ValidatorVisitor) WithPrerequisites(prerequisites ...ast.Visitor) *ValidatorVisitor {
	v.prerequisites = append(v.prerequisites, prerequisites...)
	return v
}

// VisitRoot implements Visitor.
func (v *ValidatorVisitor) VisitRoot(g *ast.Root) *ast.Root {
	for _, prerequisite := range v.prerequisites {
		g.Accept(prerequisite)
	}
	v.errors.Add(v.validator.ValidateRoot(g))
	return v.Base.VisitRoot(g)
}

// VisitSystem implements Visitor.
func (v *ValidatorVisitor) VisitSystem(c *ast.System) *ast.System {
	v.errors.Add(v.validator.ValidateSystem(c))
	return v.Base.VisitSystem(c)
}

// VisitSystemObject implements Visitor.
func (v *ValidatorVisitor) VisitSystemObject(o *ast.SystemObject) *ast.SystemObject {
	v.errors.Add(v.validator.ValidateSystemObject(o))
	return v.Base.VisitSystemObject(o)
}

// VisitClusterRegistry implements Visitor.
func (v *ValidatorVisitor) VisitClusterRegistry(c *ast.ClusterRegistry) *ast.ClusterRegistry {
	v.errors.Add(v.validator.ValidateClusterRegistry(c))
	return v.Base.VisitClusterRegistry(c)
}

// VisitClusterRegistryObject implements Visitor.
func (v *ValidatorVisitor) VisitClusterRegistryObject(o *ast.ClusterRegistryObject) *ast.ClusterRegistryObject {
	v.errors.Add(v.validator.ValidateClusterRegistryObject(o))
	return v.Base.VisitClusterRegistryObject(o)
}

// VisitCluster implements Visitor.
func (v *ValidatorVisitor) VisitCluster(c *ast.Cluster) *ast.Cluster {
	v.errors.Add(v.validator.ValidateCluster(c))
	return v.Base.VisitCluster(c)
}

// VisitClusterObject implements Visitor.
func (v *ValidatorVisitor) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	v.errors.Add(v.validator.ValidateClusterObject(o))
	return v.Base.VisitClusterObject(o)
}

// VisitTreeNode implements Visitor.
func (v *ValidatorVisitor) VisitTreeNode(n *ast.TreeNode) *ast.TreeNode {
	v.errors.Add(v.validator.ValidateTreeNode(n))
	return v.Base.VisitTreeNode(n)
}

// VisitObject implements Visitor.
func (v *ValidatorVisitor) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	v.errors.Add(v.validator.ValidateObject(o))
	return v.Base.VisitObject(o)
}

// Error implements Visitor.
func (v *ValidatorVisitor) Error() error {
	return v.errors.Build()
}
