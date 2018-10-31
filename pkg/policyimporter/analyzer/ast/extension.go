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
	"reflect"

	"github.com/pkg/errors"
)

// Extension holds visitor specific data for iterative transform passes.  Extension handles calls to a nil
// object since it is inteded to act as an immutable data store that produces updated copies rather
// than mutating underlying state.  The interface is strict because the intent is to allow users to
// extend AST Nodes without cluttering them with additional member fields.  Given that this is
// effectively implementing runtime extension of member variables, any operation failure is a sign
// of programmer error not other causes.  The suggested usage pattern is for writers to Add or Update once
// during the tree visit, and for readers to Get once during the tree visit.  Any objects that hold
// mutable state that is intended to change over the course of the visit should exist on the visitor
// itself.
// This is useful for implementing transforms in multiple passes where a single pass would either be
// not feasible or overly complicated.  An example is the two-pass label based policy transform
// that first builds a map of namespace selectors to objects in one pass, then in another it inlines
// the objects referencing elements of the map.
// Example Usage:
//	type myKey struct{}
//	type myContext struct {
//		// ... my field
//	}
//
//	func (v *MyVisitor) VisitRoot(o Root) Node {
//		v.ctx := &myContext{}
//		v.Base.VisitRoot(o)
//		o.Extension.Add(myKey{}, v.ctx)
//		return o
//	}
type Extension struct {
	items map[interface{}]interface{}
}

// newExtension returns a new data object.
func newExtension() *Extension {
	return &Extension{
		items: map[interface{}]interface{}{},
	}
}

// Equal returns true if e and other are exactly equal.  Used in the "cmp" test
// comparisons.
func (d *Extension) Equal(other *Extension) bool {
	if d == nil && other != nil {
		return false
	}
	if d != nil && other == nil {
		return false
	}
	if d == nil && other == nil {
		return true
	}
	return reflect.DeepEqual(d.items, other.items)
}

// Copy creates a copy of Extension.  If data is nil, it will return nil.
func (d *Extension) Copy() *Extension {
	if d == nil {
		return nil
	}
	nd := newExtension()
	for k, v := range d.items {
		nd.items[k] = v
	}
	return nd
}

// Len returns the number of items in Extension.
func (d *Extension) Len() int {
	if d == nil {
		return 0
	}
	return len(d.items)
}

// Add creates a copy of Extension with the new item.  If the key already exists, Add will panic.
func (d *Extension) Add(key, value interface{}) *Extension {
	if d == nil {
		return newExtension().Add(key, value)
	}
	if v, found := d.items[key]; found {
		panic(errors.Errorf(
			"programmer error: key %s already added with value %#v while trying to add %#v", key, v, value))
	}

	nd := d.Copy()
	nd.items[key] = value
	return nd
}

// Update creates a copy of Extension with a new value for an existing key.  If the key does
// not exist, Update will panic.
func (d *Extension) Update(key, value interface{}) *Extension {
	if d == nil {
		return newExtension().Update(key, value)
	}
	if _, found := d.items[key]; !found {
		panic(errors.Errorf(
			"programmer error: key %s does not exist when trying to update to value %#v", key, value))
	}
	nd := d.Copy()
	nd.items[key] = value
	return nd
}

// Get returns the data from the object.  If the key does not exist, Get will panic.
func (d *Extension) Get(key interface{}) interface{} {
	if d == nil {
		return newExtension().Get(key)
	}
	value, found := d.items[key]
	if !found {
		panic(errors.Errorf(
			"programmer error: key %s does not exist, unable to get value", key))
	}
	return value
}

// Remove removes the data from the object, returns an updated copy and returns a boolean to indicate
// if the value existed.  If the value did not exist in the map, Remove will panic.  If the last
// item is deleted from Extension, then nil will be returned.
func (d *Extension) Remove(key interface{}) *Extension {
	if d == nil {
		return newExtension().Remove(key)
	}
	if _, found := d.items[key]; !found {
		panic(errors.Errorf(
			"programmer error: unable to delete key %s, does not exist in map", key))
	}
	if len(d.items) == 1 {
		return nil
	}
	nd := d.Copy()
	delete(nd.items, key)
	return nd
}
