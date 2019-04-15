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
	"github.com/pkg/errors"
)

type key1Type struct{}
type key2Type struct{}

var key1 key1Type
var key2 key2Type

var value1 = "value1"
var value2 = "value2"

func TestAdd(t *testing.T) {
	testCases := []struct {
		name       string
		extension  *Extension
		addKey     interface{}
		shouldFail bool
	}{
		{
			name:   "set on nil works",
			addKey: key1,
		},
		{
			name: "set duplicate key returns error",
			extension: &Extension{items: map[interface{}]interface{}{
				key1: value1,
			}},
			addKey:     key1,
			shouldFail: true,
		},
		{
			name: "set different key works",
			extension: &Extension{items: map[interface{}]interface{}{
				key2: value1,
			}},
			addKey: key1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Add(tc.extension, tc.addKey, value1)

			if tc.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(errors.Wrap(err, "unexpected error"))
			}
		})
	}
}

func TestGet(t *testing.T) {
	testCases := []struct {
		name        string
		extension   *Extension
		getKey      interface{}
		expectValue interface{}
		shouldFail  bool
	}{
		{
			name:       "get from nil returns error",
			getKey:     key1,
			shouldFail: true,
		},
		{
			name: "get with key returns value",
			extension: &Extension{items: map[interface{}]interface{}{
				key1: value1,
			}},
			getKey:      key1,
			expectValue: value1,
		},
		{
			name: "get with wrong key returns error",
			extension: &Extension{items: map[interface{}]interface{}{
				key1: value1,
			}},
			getKey:     key2,
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := Get(tc.extension, tc.getKey)

			if tc.shouldFail {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(errors.Wrap(err, "unexpected error"))
			}

			if diff := cmp.Diff(tc.expectValue, val); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestAddTwice(t *testing.T) {
	// Ensure Add does not lose previously set keys.
	var extension *Extension
	extension, err := Add(extension, key1, value1)
	if err != nil {
		t.Fatal(err)
	}

	extension, err = Add(extension, key2, value2)
	if err != nil {
		t.Fatal(err)
	}

	val, err := Get(extension, key1)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(value1, val); diff != "" {
		t.Fatal(diff)
	}

	val2, err := Get(extension, key2)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(value2, val2); diff != "" {
		t.Fatal(diff)
	}
}
