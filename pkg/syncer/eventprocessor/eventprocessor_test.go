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
// Reviewed by sunilarora

package eventprocessor

import (
	"reflect"
	"testing"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/nomos/v1"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
)

type FakeHierarchy struct {
	subtreeName string
	subtree     []string
	subtreeErr  error
}

func (s *FakeHierarchy) Ancestry(name string) (hierarchy.Ancestry, error) {
	return nil, nil
}

func (s *FakeHierarchy) Subtree(name string) ([]string, error) {
	s.subtreeName = name
	return s.subtree, s.subtreeErr
}

func setup(subtree []string, err error) (*FakeHierarchy, *PolicyNodeEventProcessor) {
	fh := &FakeHierarchy{
		subtree:    subtree,
		subtreeErr: err,
	}
	ep := &PolicyNodeEventProcessor{
		queue:     workqueue.New(),
		hierarchy: fh,
	}
	return fh, ep
}

func getQueueElements(queue workqueue.Interface) []string {
	queue.ShutDown()
	subtree := []string{}
	for {
		item, done := queue.Get()
		if done {
			break
		}
		subtree = append(subtree, item.(string))
	}
	return subtree
}

func TestAdd(t *testing.T) {
	fh, ep := setup([]string{"a", "b", "c", "d"}, nil)

	name := "foobar"
	ep.OnAdd(&policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: name}})

	if name != fh.subtreeName {
		t.Errorf("Expected lookup for name %s got %s", name, fh.subtreeName)
	}
	elts := getQueueElements(ep.queue)
	if !reflect.DeepEqual(fh.subtree, elts) {
		t.Errorf("Expected subtree %s got %s", fh.subtree, elts)
	}
}

func TestUpdate(t *testing.T) {
	fh, ep := setup([]string{"a", "b", "c", "d", "e"}, nil)

	name := "foobar"
	oldNode := &policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: name}}
	newNode := &policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: name}}
	ep.OnUpdate(oldNode, newNode)
	if name != fh.subtreeName {
		t.Errorf("Expected lookup for name %s got %s", name, fh.subtreeName)
	}
	elts := getQueueElements(ep.queue)
	if !reflect.DeepEqual(fh.subtree, elts) {
		t.Errorf("Expected subtree %s got %s", fh.subtree, elts)
	}
}

func TestError(t *testing.T) {
	name := "foobar"
	fh, ep := setup([]string{}, &hierarchy.NotFoundError{})

	ep.OnAdd(&policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: name}})
	if name != fh.subtreeName {
		t.Errorf("Expected lookup for name %s got %s", name, fh.subtreeName)
	}
	elts := getQueueElements(ep.queue)
	if len(elts) != 0 {
		t.Errorf("Expected no elements in subtree")
	}
}

func TestDelete(t *testing.T) {
	_, ep := setup([]string{"a", "b", "c", "d", "e"}, nil)

	name := "foobar"
	ep.OnDelete(&policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: name}})
	expect := []string{name}
	elts := getQueueElements(ep.queue)
	if !reflect.DeepEqual(expect, elts) {
		t.Errorf("Expected subtree %s got %s", expect, elts)
	}
}
