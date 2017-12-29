/*
Copyright 2017 The Stolos Authors.

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

package fakeorg

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func source(skip int) string {
	_, file, line, _ := runtime.Caller(skip + 1)
	return fmt.Sprintf("\n%s:%d", filepath.Base(file), line)
}

type TH struct {
	t *testing.T
}

func (th *TH) logError(expected interface{}, value interface{}, values ...interface{}) {
	th.t.Error(
		append(
			[]interface{}{fmt.Sprintf("%s: \nExpected: %sActual: %s",
				source(2),
				spew.Sdump(expected),
				spew.Sdump(value))},
			values...))
}

func (th *TH) logErrorf(expected interface{}, value interface{}, msg string, values ...interface{}) {
	th.logErrorf(
		"%s: \nExpected: %sActual: %s%s",
		source(2),
		spew.Sdump(expected),
		spew.Sdump(value),
		fmt.Sprintf(msg, values...))
}

func (th *TH) ExpectTrue(value bool, values ...interface{}) {
	if !value {
		th.logError(true, value, values...)
	}
}

func (th *TH) ExpectFalse(value bool, values ...interface{}) {
	if value {
		th.logError(false, value, values...)
	}
}

func (th *TH) ExpectEQ(expected interface{}, value interface{}, values ...interface{}) {
	if !reflect.DeepEqual(expected, value) {
		th.logError(expected, value, values...)
	}
}

func strSort(values []string) []string {
	var sorted sort.StringSlice
	sorted = values
	sort.Sort(sorted)
	return sorted
}

func TestNodeOperations(t *testing.T) {
	th := &TH{t: t}

	rootNode := NewNode("root")
	th.ExpectTrue(rootNode.IsLeaf(), "root should be leaf")
	th.ExpectTrue(rootNode.IsRoot(), "root node should be root")

	childNode := NewNode("child")
	rootNode.addChild(childNode)
	th.ExpectFalse(rootNode.IsLeaf(), "root should not be leaf")
	th.ExpectTrue(childNode.IsLeaf(), "Child should be leaf")
	th.ExpectTrue(rootNode.IsRoot(), "root node should be root")
	th.ExpectFalse(childNode.IsRoot(), "child node should not be root")
	th.ExpectTrue(rootNode.IsDescendant(childNode), "child descends from root")
	th.ExpectFalse(childNode.IsDescendant(rootNode), "root does not descend from child")
	th.ExpectEQ([]string{"child", "root"}, strSort(rootNode.Subtree()), "Root nodd descendants")
	th.ExpectEQ([]string{"child"}, strSort(childNode.Subtree()), "Root node descendants")
	th.ExpectEQ([]string{"child"}, strSort(rootNode.Leaves()), "Root node invalid leaves")
	th.ExpectEQ([]string{"child"}, strSort(childNode.Leaves()), "Child node invalid leaves")

	childNode2 := NewNode("child2")
	childNode.addChild(childNode2)
	th.ExpectEQ([]string{"child", "child2", "root"}, strSort(rootNode.Subtree()), "Root node descendants")
	th.ExpectEQ([]string{"child", "child2"}, strSort(childNode.Subtree()), "Root node descendants")
	th.ExpectFalse(rootNode.IsDescendant(rootNode), "root should not descend from root")
	th.ExpectTrue(rootNode.IsDescendant(childNode), "child should descend from root")
	th.ExpectTrue(rootNode.IsDescendant(childNode2), "child2 should descend from root")
	th.ExpectFalse(childNode2.IsDescendant(rootNode), "child2 should descend from root")
	th.ExpectEQ([]string{"child2"}, strSort(rootNode.Leaves()), "Root node invalid leaves")
	th.ExpectEQ([]string{"child2"}, strSort(childNode.Leaves()), "Child node invalid leaves")

	rootNode.removeChild(childNode)
	th.ExpectTrue(rootNode.IsRoot(), "root node should be root")
	th.ExpectTrue(rootNode.IsLeaf(), "root should be leaf")
	th.ExpectEQ([]string{"root"}, strSort(rootNode.Subtree()), "Root nodd descendants")
	th.ExpectEQ([]string{"root"}, strSort(rootNode.Leaves()), "Root node invalid leaves")

	th.ExpectTrue(childNode.IsRoot(), "child node should be root")
	th.ExpectEQ([]string{"child", "child2"}, strSort(childNode.Subtree()), "Root node descendants")
	th.ExpectEQ([]string{"child2"}, strSort(childNode.Leaves()), "Child node invalid leaves")
}
