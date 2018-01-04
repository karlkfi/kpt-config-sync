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

package syncer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/syncer/actions"
	actions_testing "github.com/google/stolos/pkg/syncer/actions/testing"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testQueue struct {
	items []interface{}
}

func (s *testQueue) Add(item interface{}) {
	s.items = append(s.items, item)
}

// testUnpackerInterfaceImpl implements the UnpackerInterface and allows for mocking out return
// values from the interface function calls.
type testUnpackerInterfaceImpl struct {
	t *testing.T

	// Return values
	updateRemovalsRet []interface{}
	upsertRet         []interface{}
	namesRet          map[string]bool
}

var _ UnpackerInterface = &testUnpackerInterfaceImpl{}

func newTestResourceInterfaceImpl(t *testing.T) *testUnpackerInterfaceImpl {
	return &testUnpackerInterfaceImpl{
		t:        t,
		namesRet: map[string]bool{},
	}
}

func (s *testUnpackerInterfaceImpl) UpdateRemovals(
	old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) []interface{} {
	return s.updateRemovalsRet
}

func (s *testUnpackerInterfaceImpl) Upserts(node *policyhierarchy_v1.PolicyNode) []interface{} {
	return s.upsertRet
}

func (s *testUnpackerInterfaceImpl) Names(node *policyhierarchy_v1.PolicyNode) map[string]bool {
	return s.namesRet
}

func CheckQueueActions(t *testing.T, items []interface{}, expected []string) {
	got := []string{}
	for i := 0; i < len(items); i++ {
		got = append(got, items[i].(actions.Interface).String())
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("Items and expected not equal: %v\ngot: %v\nexpected: %v",
			cmp.Diff(got, expected), got, expected)
	}
}

func TestOnCreate(t *testing.T) {
	testResourceImpl := newTestResourceInterfaceImpl(t)
	testActionImpl := actions_testing.NewTestResourceInterfaceImpl(t)
	queue := &testQueue{}
	node := &policyhierarchy_v1.PolicyNode{}

	g := NewGenericSyncer(testActionImpl, testResourceImpl, queue)

	testResourceImpl.upsertRet = []interface{}{
		actions_testing.NewTestResourceType("foo", "baz"),
		actions_testing.NewTestResourceType("foo", "bar"),
	}
	err := g.OnCreate(node)
	if err != nil {
		t.Errorf("Create should not have failed: %s", err)
	}
	CheckQueueActions(t, queue.items, []string{
		"testresource.foo.baz.upsert",
		"testresource.foo.bar.upsert"})
}

func TestOnUpdate(t *testing.T) {
	testResourceImpl := newTestResourceInterfaceImpl(t)
	testActionImpl := actions_testing.NewTestResourceInterfaceImpl(t)
	queue := &testQueue{}
	node := &policyhierarchy_v1.PolicyNode{}

	g := NewGenericSyncer(testActionImpl, testResourceImpl, queue)

	testResourceImpl.upsertRet = []interface{}{
		actions_testing.NewTestResourceType("foo", "baz"),
		actions_testing.NewTestResourceType("foo", "bar"),
	}
	testResourceImpl.updateRemovalsRet = []interface{}{
		actions_testing.NewTestResourceType("foo", "foo"),
	}
	err := g.OnUpdate(node, node)
	if err != nil {
		t.Errorf("Create should not have failed: %s", err)
	}
	CheckQueueActions(t, queue.items, []string{
		"testresource.foo.foo.delete",
		"testresource.foo.baz.upsert",
		"testresource.foo.bar.upsert"})
}

func TestPeriodicResync(t *testing.T) {
	testResourceImpl := newTestResourceInterfaceImpl(t)
	testActionImpl := actions_testing.NewTestResourceInterfaceImpl(t)
	queue := &testQueue{}
	node := &policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: "foo"}}

	g := NewGenericSyncer(testActionImpl, testResourceImpl, queue)

	testResourceImpl.namesRet = map[string]bool{
		"res1": true,
		"res2": true,
		"res3": true,
	}
	testActionImpl.ValuesRet = map[string]interface{}{
		"res1": nil,
		"res2": nil,
		"res4": actions_testing.NewTestResourceType("foo", "res4"),
	}

	err := g.PeriodicResync([]*policyhierarchy_v1.PolicyNode{
		node,
	})
	if err != nil {
		t.Errorf("Create should not have failed: %s", err)
	}
	CheckQueueActions(t, queue.items, []string{"testresource.foo.res4.delete"})
}
