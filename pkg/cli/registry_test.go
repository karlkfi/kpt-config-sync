/*
Copyright 2017 The Nomos Authors.

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

package cli

import (
	"reflect"
	"testing"
)

func testCallback(ctx *CommandContext, args []string) error {
	return nil
}

func TestX(t *testing.T) {
	r := newRegistryNode()
	err := r.addCommand([]string{"foo", "bar"}, testCallback)
	if err != nil {
		t.Errorf("Should not have returned error")
	}
	err = r.addCommand([]string{"foo", "bar"}, testCallback)
	if err == nil {
		t.Errorf("Should have returned error")
	}

	args := []string{"foo", "bar", "baz", "wat"}
	for i := 2; i < 4; i++ {
		curArgs := args[:i]
		expectedRest := args[2:i]
		node, rest := r.getNodeByPrefix(curArgs)
		if node == nil {
			t.Errorf("Should have returned node")
		}
		if !reflect.DeepEqual(expectedRest, rest) {
			t.Errorf("Should have returned %#v, got %#v", expectedRest, rest)
		}
	}

	node := r.getNode([]string{"foo"})
	if node == nil {
		t.Errorf("Should not be nil")
	}
	if node.function != nil {
		t.Errorf("Should not have set function in foo")
	}
	node = r.getNode([]string{"foo", "bar"})
	if node == nil {
		t.Errorf("Should not be nil")
	}
	if node.function == nil {
		t.Errorf("Should have set function in foo bar")
	}
	node = r.getNode([]string{"foo", "bar", "baz"})
	if node != nil {
		t.Errorf("Should be nil")
	}

	node = r.getOrCreateNode([]string{"foo", "bar"})
	if node == nil {
		t.Errorf("Should not be nil")
	}
	if node.function == nil {
		t.Errorf("Should have set function in foo bar")
	}

	node = r.getOrCreateNode([]string{"some", "other", "command"})
	if node == nil {
		t.Errorf("Should not be nil")
	}
	if node.function != nil {
		t.Errorf("Should not have set function")
	}
}
