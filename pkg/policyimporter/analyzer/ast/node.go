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

import "fmt"

// NodeType represents the type of the node.
type NodeType string

const (
	// Namespace represents a kubernetes namespace
	Namespace = NodeType("namespace")
	// Policyspace represents a nomos policy space
	Policyspace = NodeType("policyspace")
)

// Node is analgous to a directory in the policy hierarchy.
type Node struct {
	// Path is the path to the current directory from the root of the tree, may be useful
	// for adding annotations or other things.
	Path string

	// The type of the Node
	Type        NodeType
	Labels      map[string]string
	Annotations map[string]string

	// Objects from the directory
	Objects []*Object

	// children of the directory
	Children []*Node
}

// Accept implements Visitable
func (n *Node) Accept(visitor Visitor) {
	visitor.VisitNode(n)
}

// UnlinkedCopy returns a shallow copy of the current struct.
func (n Node) UnlinkedCopy() *Node {
	n.Objects = nil
	n.Children = nil
	return &n
}

// AddChild implements Visitable
func (n *Node) AddChild(v Visitable) {
	switch child := v.(type) {
	case *Object:
		n.Objects = append(n.Objects, child)
	case *Node:
		n.Children = append(n.Children, child)
	default:
		panic(fmt.Sprintf("invalid child type for Cluster: %#v", child))
	}
}
