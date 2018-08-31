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
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nodeContext keeps track of objects during the tree traversal for purposes of inheriting values.
type nodeContext struct {
	nodeType  ast.TreeNodeType // the type of node being processed
	nodePath  string           // the node's path, used for annotating inherited objects
	inherited []*ast.Object    // the objects that are inherited from the node.
}

// InheritanceVisitor aggregates hierarchical quota.
type InheritanceVisitor struct {
	// cv is used for copying parts of the ast.Context tree and continuing underlying visitor iteration.
	cv *visitor.Copying
	// groupKinds contains the set of GroupKind that will be targeted during the inheritance transform.
	groupKinds map[schema.GroupKind]bool
	// treeContext is a stack that tracks ancestry and inherited objects during the tree traversal.
	treeContext []nodeContext
}

// NewInheritanceVisitor returns a new InheritanceVisitor for the given GroupKind
func NewInheritanceVisitor(resources []schema.GroupKind) *InheritanceVisitor {
	resourceMap := map[schema.GroupKind]bool{}
	for _, r := range resources {
		resourceMap[r] = true
	}
	cv := visitor.NewCopying()
	iv := &InheritanceVisitor{
		cv:         cv,
		groupKinds: resourceMap,
	}
	cv.SetImpl(iv)
	return iv
}

// Result implements MutatingVisitor
func (v *InheritanceVisitor) Result() error {
	return nil
}

// VisitContext implements Visitor
func (v *InheritanceVisitor) VisitContext(g *ast.Context) ast.Node {
	return v.cv.VisitContext(g)
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
	newNode := v.cv.VisitTreeNode(n).(*ast.TreeNode)
	v.treeContext = v.treeContext[:len(v.treeContext)-1]
	if n.Type == ast.Namespace {
		for _, ctx := range v.treeContext {
			newNode.Objects = append(newNode.Objects, ctx.inherited...)
		}
	}
	return newNode
}

// VisitObject implements Visitor
func (v *InheritanceVisitor) VisitObject(o *ast.Object) ast.Node {
	context := &v.treeContext[len(v.treeContext)-1]
	groupKind := o.GetObjectKind().GroupVersionKind().GroupKind()
	if context.nodeType == ast.Policyspace && v.groupKinds[groupKind] {
		obj := o.DeepCopy()
		annotations := obj.ToMeta().GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[policyhierarchyv1.AnnotationKeyDeclarationDirectory] = context.nodePath
		obj.ToMeta().SetAnnotations(annotations)
		context.inherited = append(context.inherited, obj)
		return nil
	}
	return o
}
