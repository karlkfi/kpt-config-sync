/*
Copyright 2017 The Kubernetes Authors.

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

// Package fakeorg generates a fake organization hierarchy and then performs mutations on it.
package fakeorg

import "testing"

func TestFakeOrg(t *testing.T) {
	th := &TH{t: t}

	fakeOrg := New("testorg")

	th.ExpectTrue(fakeOrg.RootNode() != nil)
	th.ExpectEQ(1, fakeOrg.Len())
	th.ExpectEQ(strSort([]string{"testorg"}), fakeOrg.NodeNames())
	th.ExpectTrue(fakeOrg.Contains("testorg"))

	testNode1 := NewNode("child1")
	fakeOrg.AddNode(fakeOrg.RootNode(), testNode1)
	th.ExpectEQ(2, fakeOrg.Len())
	th.ExpectEQ([]string{"child1", "testorg"}, strSort(fakeOrg.NodeNames()))
	th.ExpectTrue(fakeOrg.Contains("child1"))
	th.ExpectTrue(fakeOrg.Contains("testorg"))

	testNode2 := NewNode("child2")
	fakeOrg.AddNode(testNode1, testNode2)
	th.ExpectEQ(3, fakeOrg.Len())
	th.ExpectEQ([]string{"child1", "child2", "testorg"}, strSort(fakeOrg.NodeNames()))
	th.ExpectEQ([]string{"child1", "child2"}, strSort(testNode1.Subtree()))
	th.ExpectTrue(fakeOrg.Contains("child1"))
	th.ExpectTrue(fakeOrg.Contains("child2"))
	th.ExpectTrue(fakeOrg.Contains("testorg"))

	fakeOrg.ReparentNode(fakeOrg.RootNode(), testNode2)
	th.ExpectEQ([]string{"child1"}, strSort(testNode1.Subtree()))
	th.ExpectEQ([]string{"child1", "child2", "testorg"}, strSort(fakeOrg.RootNode().Subtree()))
	th.ExpectEQ(3, fakeOrg.Len())
	th.ExpectEQ([]string{"child1", "child2", "testorg"}, strSort(fakeOrg.NodeNames()))
	th.ExpectTrue(fakeOrg.Contains("child1"))
	th.ExpectTrue(fakeOrg.Contains("child2"))
	th.ExpectTrue(fakeOrg.Contains("testorg"))

	fakeOrg.RemoveNode(testNode2)
	th.ExpectEQ(2, fakeOrg.Len())
	th.ExpectEQ([]string{"child1", "testorg"}, strSort(fakeOrg.NodeNames()))
	th.ExpectTrue(fakeOrg.Contains("child1"))
	th.ExpectFalse(fakeOrg.Contains("child2"))
	th.ExpectTrue(fakeOrg.Contains("testorg"))

	fakeOrg.RemoveNode(testNode1)
	th.ExpectEQ(1, fakeOrg.Len())
	th.ExpectEQ([]string{"testorg"}, strSort(fakeOrg.NodeNames()))
	th.ExpectFalse(fakeOrg.Contains("child1"))
	th.ExpectFalse(fakeOrg.Contains("child2"))
	th.ExpectTrue(fakeOrg.Contains("testorg"))
}
