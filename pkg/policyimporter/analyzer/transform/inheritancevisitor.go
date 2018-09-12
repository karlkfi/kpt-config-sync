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
	"path/filepath"

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

// InheritanceSpec defines the spec for inherited resources.
type InheritanceSpec struct {
	GroupVersionKind  schema.GroupVersionKind
	PolicyspacePrefix bool
}

// InheritanceVisitor aggregates hierarchical quota.
type InheritanceVisitor struct {
	// cv is used for copying parts of the ast.Context tree and continuing underlying visitor iteration.
	cv *visitor.Copying
	// groupKinds contains the set of GroupKind that will be targeted during the inheritance transform.
	inheritanceSpecs map[schema.GroupVersionKind]*InheritanceSpec
	// treeContext is a stack that tracks ancestry and inherited objects during the tree traversal.
	treeContext []nodeContext
}

// NewInheritanceVisitor returns a new InheritanceVisitor for the given GroupKind
func NewInheritanceVisitor(resources []InheritanceSpec) *InheritanceVisitor {
	resourceMap := map[schema.GroupVersionKind]*InheritanceSpec{}
	for idx := range resources {
		r := &resources[idx]
		resourceMap[r.GroupVersionKind] = r
	}
	cv := visitor.NewCopying()
	iv := &InheritanceVisitor{
		cv:               cv,
		inheritanceSpecs: resourceMap,
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

// VisitObjectList implements Visitor
func (v *InheritanceVisitor) VisitObjectList(o ast.ObjectList) ast.Node {
	return v.cv.VisitObjectList(o)
}

// VisitObject implements Visitor
func (v *InheritanceVisitor) VisitObject(o *ast.Object) ast.Node {
	context := &v.treeContext[len(v.treeContext)-1]
	gvk := o.GetObjectKind().GroupVersionKind()
	if spec, found := v.inheritanceSpecs[gvk]; context.nodeType == ast.Policyspace && found {
		if spec.PolicyspacePrefix {
			o = o.DeepCopy()
			meta := o.ToMeta()
			meta.SetName(filepath.Base(context.nodePath) + "." + meta.GetName())
		}
		context.inherited = append(context.inherited, o)
		return nil
	}
	return o
}
