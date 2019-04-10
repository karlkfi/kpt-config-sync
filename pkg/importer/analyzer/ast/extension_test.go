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
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	add    = "add"
	update = "update"
	get    = "get"
	remove = "remove"
)

var cmpOpt = cmp.AllowUnexported(Extension{})

type key1Type struct{}
type key2Type struct{}

var key1 key1Type
var key2 key2Type

type extensionOperation struct {
	typ  string
	key  interface{}
	item string

	expectPanic bool
	expectItem  string
}

type extensionTestcase struct {
	name string
	ops  []extensionOperation
}

func (tc *extensionTestcase) runOp(t *testing.T, d *Extension, idx int) *Extension {
	op := tc.ops[idx]
	defer func() {
		if v := recover(); v != nil {
			if !op.expectPanic {
				t.Fatalf("idx %d: did not expect panic, got panic: %v", idx, v)
			}
		}
	}()

	var nd *Extension
	var val interface{}

	inCopy := d.Copy()
	switch op.typ {
	case add:
		nd = d.Add(op.key, op.item)
	case update:
		nd = d.Update(op.key, op.item)
	case get:
		val = d.Get(op.key)
	case remove:
		nd = d.Remove(op.key)
	default:
		t.Fatalf("invalid op type: %s", op.typ)
	}
	if !cmp.Equal(d, inCopy, cmpOpt) {
		t.Errorf("Extension was modified during operation %s", cmp.Diff(d, inCopy, cmpOpt))
	}
	if op.expectPanic {
		// we should not get here.
		t.Fatalf("Expected panic.")
	}

	if op.typ == get {
		if !cmp.Equal(val, op.expectItem, cmpOpt) {
			t.Errorf("idx %d: Wrong item returned: %s", idx, cmp.Diff(op.expectItem, val, cmpOpt))
		}
		return d
	}
	if nd == nil && nd != nil {
		t.Errorf("data should be nil if emtpy")
	}

	return nd
}

func (tc *extensionTestcase) Run(t *testing.T) {
	var d *Extension
	for idx := range tc.ops {
		d = tc.runOp(t, d, idx)
	}
}

var extensionTestcases = []extensionTestcase{
	{
		name: "Add get update get remove get",
		ops: []extensionOperation{
			{
				typ:  add,
				key:  key1,
				item: "value1",
			},
			{
				typ:        get,
				key:        key1,
				expectItem: "value1",
			},
			{
				typ:  update,
				key:  key1,
				item: "value2",
			},
			{
				typ:        get,
				key:        key1,
				expectItem: "value2",
			},
			{
				typ: remove,
				key: key1,
			},
			{
				typ:         get,
				key:         key1,
				expectPanic: true,
			},
		},
	},
	{
		name: "Add existing value",
		ops: []extensionOperation{
			{
				typ:  add,
				key:  key1,
				item: "value1",
			},
			{
				typ:        get,
				key:        key1,
				expectItem: "value1",
			},
			{
				typ:         add,
				key:         key1,
				item:        "value2",
				expectPanic: true,
			},
		},
	},
	{
		name: "Update non existent value",
		ops: []extensionOperation{
			{
				typ:         update,
				key:         key1,
				item:        "value",
				expectPanic: true,
			},
		},
	},
	{
		name: "get not found",
		ops: []extensionOperation{
			{
				typ:         get,
				key:         key1,
				expectPanic: true,
			},
		},
	},
	{
		name: "Remove non existent value",
		ops: []extensionOperation{
			{
				typ:         remove,
				key:         key1,
				expectPanic: true,
			},
		},
	},
	{
		name: "Handle two struct keys",
		ops: []extensionOperation{
			{
				typ:  add,
				key:  key1,
				item: "key1-value",
			},
			{
				typ:  add,
				key:  key2,
				item: "key2-value",
			},
			{
				typ:        get,
				key:        key1,
				expectItem: "key1-value",
			},
			{
				typ:        get,
				key:        key2,
				expectItem: "key2-value",
			},
		},
	},
}

func TestExtension(t *testing.T) {
	for _, tc := range extensionTestcases {
		t.Run(tc.name, tc.Run)
	}
}
