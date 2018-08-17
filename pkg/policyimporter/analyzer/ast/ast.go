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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Context represents a hierarchy of kubernetes policies.
type Context struct {
	ImportToken string    // Import token for context
	LoadTime    time.Time // Time at which the context was generated

	ReservedNamespaces *ReservedNamespaces // Reserved namespaces
	Cluster            *Cluster            // Cluster scoped info
	Tree               *Node               // Hirearchical policies
}

// Accept implements Visitable
func (g *Context) Accept(visitor Visitor) {
	visitor.VisitContext(g)
}

// UnlinkedCopy returns a shallow copy of the current struct.  Slices and Maps contained as members
// of the struct will be copied as to prevent issues on update.
func (g Context) UnlinkedCopy() *Context {
	g.Cluster = nil
	g.ReservedNamespaces = nil
	g.Tree = nil
	return &g
}

// AddChild implements Visitable
func (g *Context) AddChild(child Visitable) {
	switch c := child.(type) {
	case *Cluster:
		if g.Cluster != nil {
			panic("cluster already specified")
		}
		g.Cluster = c
	case *Node:
		if g.Tree != nil {
			panic("node already specified")
		}
		g.Tree = c
	case *ReservedNamespaces:
		if g.ReservedNamespaces != nil {
			panic("node already specified")
		}
		g.ReservedNamespaces = c
	default:
		panic(fmt.Sprintf("invalid child type for GitContext: %#v", child))
	}
}

// Cluster represents cluster scoped policies.
type Cluster struct {
	Objects []*Object
}

// Accept implements Visitable
func (c *Cluster) Accept(visitor Visitor) {
	visitor.VisitCluster(c)
}

// UnlinkedCopy returns a shallow copy of the current struct.
func (c Cluster) UnlinkedCopy() *Cluster {
	c.Objects = nil
	return &c
}

// AddChild implements Visitable
func (c *Cluster) AddChild(v Visitable) {
	switch child := v.(type) {
	case *Object:
		c.Objects = append(c.Objects, child)
	default:
		panic(fmt.Sprintf("invalid child type for Cluster: %#v", child))
	}
}

// Object extends runtime.Object to implement Visitable.
type Object struct {
	runtime.Object
}

// ToMeta converts the underlying object to a metav1.Object
func (o *Object) ToMeta() metav1.Object {
	return o.Object.(metav1.Object)
}

// Accept implements Visitable
func (o *Object) Accept(visitor Visitor) {
	visitor.VisitObject(o)
}

// DeepCopy creates a deep copy of the object
func (o *Object) DeepCopy() *Object {
	return &Object{o.DeepCopyObject()}
}

// AddChild implements Visitable
func (o *Object) AddChild(child Visitable) {
	panic("children cannot be added to Object")
}

// ReservedNamespaces represents the reserved namespaces object
type ReservedNamespaces struct {
	corev1.ConfigMap
}

// Accept implements Visitable
func (r *ReservedNamespaces) Accept(visitor Visitor) {
	visitor.VisitReservedNamespaces(r)
}

// DeepCopy creates a deep copy of ReservedNamespaces
func (r *ReservedNamespaces) DeepCopy() *ReservedNamespaces {
	return &ReservedNamespaces{*r.ConfigMap.DeepCopy()}
}

// AddChild implements Visitable
func (r *ReservedNamespaces) AddChild(child Visitable) {
	panic("children cannot be added to ReservedNamespaces")
}
