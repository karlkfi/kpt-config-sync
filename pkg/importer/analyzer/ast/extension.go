/*
Copyright 2018 The CSP Config Management Authors.

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
	"github.com/google/nomos/pkg/status"
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
		items: make(map[interface{}]interface{}),
	}
}

// Add returns a shallow copy of Extension with key set to value, or an error if the key is already
// present.
func Add(d *Extension, key, value interface{}) (*Extension, status.Error) {
	nd := newExtension()
	if d != nil {
		// Ensure key does not already exist.
		_, found := d.items[key]
		if found {
			return nil, status.InternalError("key %#v already present in Extension")
		}

		// Copy items from d into new Extension.
		for k, v := range d.items {
			nd.items[k] = v
		}
	}

	nd.items[key] = value
	return nd, nil
}

// Get returns the value at the requested key, or an error if the key is not present.
func Get(d *Extension, key interface{}) (interface{}, status.Error) {
	if d == nil {
		return nil, status.InternalError("tried to get %#v from nil Extension")
	}

	value, found := d.items[key]
	if !found {
		return nil, status.InternalError("extension missing key %#v")
	}
	return value, nil
}

// Add creates a copy of Extension with key set to value.
//
// Deprecated: Use ast.Add.
func (d *Extension) Add(key, value interface{}) *Extension {
	result, err := Add(d, key, value)
	if err != nil {
		panic(err)
	}
	return result
}

// Get returns the data from the object.  If the key does not exist, Get will panic.
//
// Deprecated: Use ast.Get.
func (d *Extension) Get(key interface{}) interface{} {
	value, err := Get(d, key)
	if err != nil {
		panic(err)
	}
	return value
}
