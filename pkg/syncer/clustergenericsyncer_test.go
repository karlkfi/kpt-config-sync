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

package syncer

import (
	"testing"

	"github.com/google/nomos/pkg/syncer/labeling"

	"github.com/google/go-cmp/cmp"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/action/test"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type testClusterPolicyUnpacker struct {
	upsertRet         []runtime.Object
	updateRemovalsRet []runtime.Object
	namesRet          map[string]bool
	listRet           []runtime.Object
	listErr           error
}

func (s *testClusterPolicyUnpacker) UpdateRemovals(
	old *policyhierarchy_v1.ClusterPolicy, new *policyhierarchy_v1.ClusterPolicy) []runtime.Object {
	return s.updateRemovalsRet
}
func (s *testClusterPolicyUnpacker) Upserts(
	node *policyhierarchy_v1.ClusterPolicy) []runtime.Object {
	return s.upsertRet
}
func (s *testClusterPolicyUnpacker) Names(
	node *policyhierarchy_v1.ClusterPolicy) map[string]bool {
	return s.namesRet
}
func (s *testClusterPolicyUnpacker) List() ([]runtime.Object, error) {
	return s.listRet, s.listErr
}
func (s *testClusterPolicyUnpacker) NewDeleteAction(name string) action.Interface {
	return test.NewDelete("", name, "ClusterRole")
}
func (s *testClusterPolicyUnpacker) NewUpsertAction(name string, obj runtime.Object) action.Interface {
	return test.NewUpsert("", name, "ClusterRole")
}

type testQueue struct {
	items []interface{}
}

func (s *testQueue) Add(item interface{}) {
	s.items = append(s.items, item)
}

func testListRet() []runtime.Object {
	return []runtime.Object{
		&rbac_v1.ClusterRole{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: "custom",
			},
		},
		&rbac_v1.ClusterRole{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   "unaccounted",
				Labels: labeling.AddOriginLabelToMap(nil),
			},
		},
	}
}

func TestGCSOnCreate(t *testing.T) {
	testUnpacker := &testClusterPolicyUnpacker{
		upsertRet: []runtime.Object{
			&rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "foo",
				},
			},
			&rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "bar",
				},
			},
		},
		listRet: testListRet(),
	}
	queue := &testQueue{}
	cgs := NewClusterGenericSyncer(testUnpacker, queue)
	cp := &policyhierarchy_v1.ClusterPolicy{}
	err := cgs.OnCreate(cp)
	if err != nil {
		t.Errorf("Create should not have failed")
	}
	CheckQueueActions(t, queue.items, []string{
		"group/v1/ClusterRole/unaccounted/delete",
		"group/v1/ClusterRole/foo/upsert",
		"group/v1/ClusterRole/bar/upsert",
	})
}

func TestGCSOnUpdate(t *testing.T) {
	testUnpacker := &testClusterPolicyUnpacker{
		updateRemovalsRet: []runtime.Object{
			&rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "baz",
				},
			},
		},
		upsertRet: []runtime.Object{
			&rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "foo",
				},
			},
			&rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "bar",
				},
			},
		},
		listRet: testListRet(),
	}
	queue := &testQueue{}
	cgs := NewClusterGenericSyncer(testUnpacker, queue)
	oldCp := &policyhierarchy_v1.ClusterPolicy{}
	newCp := &policyhierarchy_v1.ClusterPolicy{}
	err := cgs.OnUpdate(oldCp, newCp)
	if err != nil {
		t.Errorf("Create should not have failed")
	}
	CheckQueueActions(t, queue.items, []string{
		"group/v1/ClusterRole/unaccounted/delete",
		"group/v1/ClusterRole/baz/delete",
		"group/v1/ClusterRole/foo/upsert",
		"group/v1/ClusterRole/bar/upsert",
	})
}

func TestGCSOnDelete(t *testing.T) {
	testUnpacker := &testClusterPolicyUnpacker{
		updateRemovalsRet: []runtime.Object{
			&rbac_v1.ClusterRole{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "baz",
				},
			},
		},
		listRet: testListRet(),
	}
	queue := &testQueue{}
	cgs := NewClusterGenericSyncer(testUnpacker, queue)
	cp := &policyhierarchy_v1.ClusterPolicy{}
	err := cgs.OnDelete(cp)
	if err != nil {
		t.Errorf("Create should not have failed")
	}
	CheckQueueActions(t, queue.items, []string{
		"group/v1/ClusterRole/unaccounted/delete",
		"group/v1/ClusterRole/baz/delete",
	})
}

func CheckQueueActions(t *testing.T, items []interface{}, expected []string) {
	got := []string{}
	for i := 0; i < len(items); i++ {
		got = append(got, items[i].(action.Interface).String())
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("Items and expected not equal: %v\ngot: %v\nexpected: %v",
			cmp.Diff(got, expected), got, expected)
	}
}
