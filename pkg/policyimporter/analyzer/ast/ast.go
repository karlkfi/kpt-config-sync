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

package ast

import (
	"time"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/policyimporter/id"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FileObject extends runtime.FileObject to include the path to the file in the repo.
type FileObject struct {
	runtime.Object
	// Relative is the path this object has relative to a nomospath.Root.
	nomospath.Relative
}

var _ id.Resource = &FileObject{}

// NewFileObject returns an ast.FileObject with the specified underlying runtime.Object and the
// designated source file.
func NewFileObject(object runtime.Object, source nomospath.Relative) FileObject {
	return FileObject{Object: object, Relative: source}
}

// MetaObject converts the underlying object to a metav1.Object
func (o *FileObject) MetaObject() metav1.Object {
	return o.Object.(metav1.Object)
}

// GroupVersionKind unambiguously defines the kind of object.
func (o *FileObject) GroupVersionKind() schema.GroupVersionKind {
	return o.GetObjectKind().GroupVersionKind()
}

// Name returns the user-defined name of the object.
func (o *FileObject) Name() string {
	return o.MetaObject().GetName()
}

// Root represents a set of declared policies, configuration for how those policies will be
// interpreted, and information regarding where those policies came from.
type Root struct {
	// ImportToken is the token for context
	ImportToken string
	LoadTime    time.Time // Time at which the context was generated
	Repo        *v1.Repo  // Nomos repo

	// Cluster represents resources that are cluster scoped.
	Cluster *Cluster

	// ClusterRegistry represents resources that are related to multi-cluster.
	ClusterRegistry *ClusterRegistry

	// System represents resources regarding nomos configuration.
	System *System

	// Tree represents the directory hierarchy containing namespace scoped resources.
	Tree *TreeNode
	Data *Extension
}

// Accept invokes VisitRoot on the visitor.
func (c *Root) Accept(visitor Visitor) *Root {
	if c == nil {
		return nil
	}
	return visitor.VisitRoot(c)
}

// System represents cluster scoped policies.
type System struct {
	Objects []*SystemObject
}

// Accept invokes VisitSystem on the visitor.
func (s *System) Accept(visitor Visitor) *System {
	if s == nil {
		return nil
	}
	return visitor.VisitSystem(s)
}

// SystemObject extends FileObject to implement Visitable for cluster scoped objects.
//
// A SystemObject represents a cluster scoped resource from the cluster directory.
type SystemObject struct {
	FileObject
}

// Accept invokes VisitSystemObject on the visitor.
func (o *SystemObject) Accept(visitor Visitor) *SystemObject {
	if o == nil {
		return nil
	}
	return visitor.VisitSystemObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *SystemObject) DeepCopy() *SystemObject {
	return &SystemObject{FileObject{Object: o.DeepCopyObject(), Relative: o.Relative}}
}

// ClusterRegistry represents cluster scoped policies.
type ClusterRegistry struct {
	Objects []*ClusterRegistryObject
}

// Accept invokes VisitClusterRegistry on the visitor.
func (c *ClusterRegistry) Accept(visitor Visitor) *ClusterRegistry {
	if c == nil {
		return nil
	}
	return visitor.VisitClusterRegistry(c)
}

// ClusterRegistryObject extends FileObject to implement Visitable for cluster scoped objects.
//
// A ClusterRegistryObject represents a cluster scoped resource from the cluster directory.
type ClusterRegistryObject struct {
	FileObject
}

// Accept invokes VisitClusterRegistryObject on the visitor.
func (o *ClusterRegistryObject) Accept(visitor Visitor) *ClusterRegistryObject {
	if o == nil {
		return nil
	}
	return visitor.VisitClusterRegistryObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *ClusterRegistryObject) DeepCopy() *ClusterRegistryObject {
	return &ClusterRegistryObject{FileObject{Object: o.DeepCopyObject(), Relative: o.Relative}}
}

// Cluster represents cluster scoped policies.
type Cluster struct {
	Objects []*ClusterObject
}

// Accept invokes VisitCluster on the visitor.
func (c *Cluster) Accept(visitor Visitor) *Cluster {
	if c == nil {
		return nil
	}
	return visitor.VisitCluster(c)
}

// ClusterObject extends FileObject to implement Visitable for cluster scoped objects.
//
// A ClusterObject represents a cluster scoped resource from the cluster directory.
type ClusterObject struct {
	FileObject
}

// Accept invokes VisitClusterObject on the visitor.
func (o *ClusterObject) Accept(visitor Visitor) *ClusterObject {
	if o == nil {
		return nil
	}
	return visitor.VisitClusterObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *ClusterObject) DeepCopy() *ClusterObject {
	return &ClusterObject{FileObject{Object: o.DeepCopyObject(), Relative: o.Relative}}
}

// TreeNode is analogous to a directory in the policy hierarchy.
type TreeNode struct {
	// Relative is the path this node has relative to a nomospath.Root.
	nomospath.Relative

	// The type of the HierarchyNode
	Type        node.Type
	Labels      map[string]string
	Annotations map[string]string

	// Objects from the directory
	Objects []*NamespaceObject

	// Selectors is a map of name to NamespaceSelector objects found at this node.
	// One or more Objects may have an annotation referring to these NamespaceSelectors by name.
	Selectors map[string]*v1.NamespaceSelector

	// Extension holds visitor specific data.
	Data *Extension

	// children of the directory
	Children []*TreeNode
}

var _ id.TreeNode = &TreeNode{}

// Accept invokes VisitTreeNode on the visitor.
func (n *TreeNode) Accept(visitor Visitor) *TreeNode {
	if n == nil {
		return nil
	}
	return visitor.VisitTreeNode(n)
}

// PartialCopy makes an almost shallow copy of n.  An "almost shallow" copy of
// TreeNode make shallow copies of Children and members that are likely
// immutable.  A  deep copy is made of mutable members like Labels and
// Annotations.
func (n *TreeNode) PartialCopy() *TreeNode {
	nn := *n
	copyMapInto(n.Annotations, &nn.Annotations)
	copyMapInto(n.Labels, &nn.Labels)
	// Not sure if Selectors should be copied the same way.
	return &nn
}

// Name returns the name of the lowest-level directory in this node's path.
func (n *TreeNode) Name() string {
	return n.Base()
}

func copyMapInto(from map[string]string, to *map[string]string) {
	if from == nil {
		return
	}
	*to = make(map[string]string)
	for k, v := range from {
		(*to)[k] = v
	}
}

// Annotated is anything that has mutable annotations.  This is a subset of
// the interface metav1.Object, and allows us to manipulate AST objects with
// the same code that operates on Kubernetes API objects, without the need to
// implement parts of metav1.Object that don't deal with annotations.
type Annotated interface {
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)
}

var _ Annotated = (*TreeNode)(nil)

// GetAnnotations returns the annotations from n.  They are mutable if not nil.
func (n *TreeNode) GetAnnotations() map[string]string {
	return n.Annotations
}

// SetAnnotations replaces the annotations on the tree node with the supplied ones.
func (n *TreeNode) SetAnnotations(a map[string]string) {
	n.Annotations = a
}

// NamespaceObject extends FileObject to implement Visitable for namespace scoped objects.
//
// An NamespaceObject represents a resource found in a directory in the policy
// hierarchy.
type NamespaceObject struct {
	FileObject
}

// Accept invokes VisitObject on the visitor.
func (o *NamespaceObject) Accept(visitor Visitor) *NamespaceObject {
	if o == nil {
		return nil
	}
	return visitor.VisitObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *NamespaceObject) DeepCopy() *NamespaceObject {
	return &NamespaceObject{FileObject{Object: o.DeepCopyObject(), Relative: o.Relative}}
}
