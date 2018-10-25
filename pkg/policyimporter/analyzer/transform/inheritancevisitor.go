/*
Copyright 2018 The Nomos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transform

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nodeContext keeps track of objects during the tree traversal for purposes of inheriting values.
type nodeContext struct {
	nodeType  ast.TreeNodeType       // the type of node being processed
	nodePath  string                 // the node's path, used for annotating inherited objects
	inherited []*ast.NamespaceObject // the objects that are inherited from the node.
}

// InheritanceSpec defines the spec for inherited resources.
type InheritanceSpec struct {
	Mode v1alpha1.HierarchyModeType
}

// InheritanceVisitor aggregates hierarchical quota.
type InheritanceVisitor struct {
	// Copying is used for copying parts of the ast.Root tree and continuing underlying visitor iteration.
	*visitor.Copying
	// groupKinds contains the set of GroupKind that will be targeted during the inheritance transform.
	inheritanceSpecs map[schema.GroupKind]*InheritanceSpec
	// treeContext is a stack that tracks ancestry and inherited objects during the tree traversal.
	treeContext []nodeContext
}

var _ ast.Visitor = &InheritanceVisitor{}

// NewInheritanceVisitor returns a new InheritanceVisitor for the given GroupKind
func NewInheritanceVisitor(specs map[schema.GroupKind]*InheritanceSpec) *InheritanceVisitor {
	iv := &InheritanceVisitor{
		Copying:          visitor.NewCopying(),
		inheritanceSpecs: specs,
	}
	iv.SetImpl(iv)
	return iv
}

// Error implements CheckingVisitor
func (v *InheritanceVisitor) Error() error {
	return nil
}

// VisitReservedNamespaces implements Visitor
func (v *InheritanceVisitor) VisitReservedNamespaces(r *ast.ReservedNamespaces) ast.Node {
	return r
}

// VisitCluster implements Visitor
func (v *InheritanceVisitor) VisitCluster(c *ast.Cluster) ast.Node {
	return c
}

// VisitTreeNode implements Visitor
func (v *InheritanceVisitor) VisitTreeNode(n *ast.TreeNode) ast.Node {
	v.treeContext = append(v.treeContext, nodeContext{
		nodeType: n.Type,
		nodePath: n.Path,
	})
	newNode := v.Copying.VisitTreeNode(n).(*ast.TreeNode)
	v.treeContext = v.treeContext[:len(v.treeContext)-1]
	if n.Type == ast.Namespace {
		for _, ctx := range v.treeContext {
			for _, inherited := range ctx.inherited {
				if sel.IsPolicyApplicableToNamespace(n.Labels, inherited.ToMeta()) {
					newNode.Objects = append(newNode.Objects, inherited)
				}
			}
		}
	}
	return newNode
}

// VisitObject implements Visitor
func (v *InheritanceVisitor) VisitObject(o *ast.NamespaceObject) ast.Node {
	context := &v.treeContext[len(v.treeContext)-1]
	gk := o.GetObjectKind().GroupVersionKind().GroupKind()
	if context.nodeType == ast.AbstractNamespace {
		spec, found := v.inheritanceSpecs[gk]
		if found && spec.Mode == v1alpha1.HierarchyModeInherit {
			context.inherited = append(context.inherited, o)
			return nil
		}
	}
	return o
}
