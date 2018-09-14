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

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Context represents a set of declared policies, configuration for how those policies will be
// interpreted, and information regarding where those policies came from.
type Context struct {
	ImportToken string    // Import token for context
	LoadTime    time.Time // Time at which the context was generated

	ReservedNamespaces *ReservedNamespaces // Reserved namespaces
	Cluster            *Cluster            // Cluster scoped info
	Tree               *TreeNode           // Hirearchical policies
}

// Accept implements Visitable
func (c *Context) Accept(visitor Visitor) Node {
	if c == nil {
		return nil
	}
	return visitor.VisitContext(c)
}

// Cluster represents cluster scoped policies.
type Cluster struct {
	Objects ObjectList
}

// Accept implements Visitable
func (c *Cluster) Accept(visitor Visitor) Node {
	if c == nil {
		return nil
	}
	return visitor.VisitCluster(c)
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
	// Path is the path to the current directory from the root of the tree.
	Path string

	// The type of the HierarchyNode
	Type        TreeNodeType
	Labels      map[string]string
	Annotations map[string]string

	// Objects from the directory
	Objects ObjectList

	Selectors map[string]*v1.NamespaceSelector

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

// ObjectList represents a set of objects.
type ObjectList []*Object

// Accept implements Visitable
func (o ObjectList) Accept(visitor Visitor) Node {
	if o == nil {
		return nil
	}
	return visitor.VisitObjectList(o)
}

// Object extends runtime.Object to implement Visitable.
//
// An Object represents a resource found in a directory in the policy
// hierarchy.
type Object struct {
	runtime.Object
}

// ToMeta converts the underlying object to a metav1.Object
func (o *Object) ToMeta() metav1.Object {
	return o.Object.(metav1.Object)
}

// Accept implements Visitable
func (o *Object) Accept(visitor Visitor) Node {
	if o == nil {
		return nil
	}
	return visitor.VisitObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *Object) DeepCopy() *Object {
	return &Object{o.DeepCopyObject()}
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
