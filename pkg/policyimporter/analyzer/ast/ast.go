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

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// FileObject extends runtime.FileObject to include the path to the file in the repo.
type FileObject struct {
	runtime.Object
	// Source is the OS-agnostic slash-separated path to the source file from the root.
	Source string
}

// Root represents a set of declared policies, configuration for how those policies will be
// interpreted, and information regarding where those policies came from.
type Root struct {
	ImportToken string                // Import token for context
	LoadTime    time.Time             // Time at which the context was generated
	Config      *v1alpha1.NomosConfig // NomosConfig

	// ReservedNamespaces corresponds to the reserved namespaces declared in the system dir.
	ReservedNamespaces *ReservedNamespaces // Reserved namespaces
	// Cluster represents resources that are cluster scoped.
	Cluster *Cluster
	// Tree represents the directory hierarchy containing namespace scoped resources.
	Tree *TreeNode
	// Extension holds visitor specific data.
	Data *Extension
}

// Accept implements Visitable
func (c *Root) Accept(visitor Visitor) Node {
	if c == nil {
		return nil
	}
	return visitor.VisitRoot(c)
}

// Cluster represents cluster scoped policies.
type Cluster struct {
	Objects ClusterObjectList
}

// Accept implements Visitable
func (c *Cluster) Accept(visitor Visitor) Node {
	if c == nil {
		return nil
	}
	return visitor.VisitCluster(c)
}

// ClusterObjectList represents a set of cluser scoped objects.
type ClusterObjectList []*ClusterObject

// Accept implements Visitable
func (o ClusterObjectList) Accept(visitor Visitor) Node {
	if o == nil {
		return nil
	}
	return visitor.VisitClusterObjectList(o)
}

// ClusterObject extends FileObject to implement Visitable for cluster scoped objects.
//
// A ClusterObject represents a cluster scoped resource from the cluster directory.
type ClusterObject struct {
	FileObject
}

// ToMeta converts the underlying object to a metav1.NamespaceObject
func (o *ClusterObject) ToMeta() metav1.Object {
	return o.FileObject.Object.(metav1.Object)
}

// Accept implements Visitable
func (o *ClusterObject) Accept(visitor Visitor) Node {
	if o == nil {
		return nil
	}
	return visitor.VisitClusterObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *ClusterObject) DeepCopy() *ClusterObject {
	return &ClusterObject{FileObject{o.DeepCopyObject(), o.Source}}
}

// TreeNodeType represents the type of the node.
type TreeNodeType string

const (
	// Namespace represents a kubernetes namespace
	Namespace = TreeNodeType("namespace")
	// Policyspace represents a nomos policy space
	Policyspace = TreeNodeType("policyspace")
)

// TreeNode is analogous to a directory in the policy hierarchy.
type TreeNode struct {
	// Path is the OS-agnostic slash-separated path to the current node from the root of the tree.
	Path string

	// The type of the HierarchyNode
	Type        TreeNodeType
	Labels      map[string]string
	Annotations map[string]string

	// Objects from the directory
	Objects ObjectList

	// Selectors is a map of name to NamespaceSelector objects found at this node.
	// One or more Objects may have an annotation referring to these NamespaceSelectors by name.
	Selectors map[string]*v1alpha1.NamespaceSelector

	// Extension holds visitor specific data.
	Data *Extension

	// children of the directory
	Children []*TreeNode
}

// Accept implements Visitable
func (n *TreeNode) Accept(visitor Visitor) Node {
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

func copyMapInto(from map[string]string, to *map[string]string) {
	if from == nil {
		return
	}
	(*to) = make(map[string]string)
	for k, v := range from {
		(*to)[k] = v
	}
}

// ObjectList represents a set of namespace scoped objects.
type ObjectList []*NamespaceObject

// Accept implements Visitable
func (o ObjectList) Accept(visitor Visitor) Node {
	if o == nil {
		return nil
	}
	return visitor.VisitObjectList(o)
}

// NamespaceObject extends FileObject to implement Visitable for namespace scoped objects.
//
// An NamespaceObject represents a resource found in a directory in the policy
// hierarchy.
type NamespaceObject struct {
	FileObject
}

// ToMeta converts the underlying object to a metav1.Object
func (o *NamespaceObject) ToMeta() metav1.Object {
	return o.FileObject.Object.(metav1.Object)
}

// Accept implements Visitable
func (o *NamespaceObject) Accept(visitor Visitor) Node {
	if o == nil {
		return nil
	}
	return visitor.VisitObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *NamespaceObject) DeepCopy() *NamespaceObject {
	return &NamespaceObject{FileObject{o.DeepCopyObject(), o.Source}}
}

// ReservedNamespaces represents the reserved namespaces object
type ReservedNamespaces struct {
	corev1.ConfigMap
}

// Accept implements Visitable
func (r *ReservedNamespaces) Accept(visitor Visitor) Node {
	if r == nil {
		return nil
	}
	return visitor.VisitReservedNamespaces(r)
}

// DeepCopy creates a deep copy of ReservedNamespaces
func (r *ReservedNamespaces) DeepCopy() *ReservedNamespaces {
	return &ReservedNamespaces{*r.ConfigMap.DeepCopy()}
}
