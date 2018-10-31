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

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
)

type FakeLister struct {
	queriedName string
	get         *v1.PolicyNode
	getErr      error
}

func (l *FakeLister) Get(name string) (*v1.PolicyNode, error) {
	l.queriedName = name
	return l.get, l.getErr
}

func (l *FakeLister) List(selector labels.Selector) ([]*v1.PolicyNode, error) {
	return nil, nil
}

func setup(name string, err error) (*FakeLister, *PolicyNodeEventProcessor) {
	pn := &v1.PolicyNode{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if name == "" {
		pn = nil
	}

	fl := &FakeLister{
		get:    pn,
		getErr: err,
	}
	ep := &PolicyNodeEventProcessor{
		queue:      workqueue.New(),
		nodeLister: fl,
	}
	return fl, ep
}

func getQueueElements(queue workqueue.Interface) []string {
	queue.ShutDown()
	items := []string{}
	for {
		item, done := queue.Get()
		if done {
			break
		}
		items = append(items, item.(string))
	}
	return items
}

func TestAdd(t *testing.T) {
	name := "foobar"
	fl, ep := setup(name, nil)

	ep.OnAdd(&v1.PolicyNode{ObjectMeta: metav1.ObjectMeta{Name: name}})

	if name != fl.queriedName {
		t.Errorf("Expected lookup for name %s got %s", name, fl.queriedName)
	}
	elts := getQueueElements(ep.queue)
	expected := []string{name}
	if !reflect.DeepEqual(expected, elts) {
		t.Errorf("Expected queue %s got %s", expected, elts)
	}
}

func TestUpdate(t *testing.T) {
	name := "foobar"
	fl, ep := setup(name, nil)

	oldNode := &v1.PolicyNode{ObjectMeta: metav1.ObjectMeta{Name: name}}
	newNode := &v1.PolicyNode{ObjectMeta: metav1.ObjectMeta{Name: name}}
	ep.OnUpdate(oldNode, newNode)
	if name != fl.queriedName {
		t.Errorf("Expected lookup for name %s got %s", name, fl.queriedName)
	}
	elts := getQueueElements(ep.queue)
	expected := []string{name}
	if !reflect.DeepEqual(expected, elts) {
		t.Errorf("Expected queue %s got %s", expected, elts)
	}
}

func TestError(t *testing.T) {
	name := "foobar"
	fl, ep := setup("", apierrors.NewNotFound(schema.GroupResource{}, "not found"))

	ep.OnAdd(&v1.PolicyNode{ObjectMeta: metav1.ObjectMeta{Name: name}})
	if name != fl.queriedName {
		t.Errorf("Expected lookup for name %s got %s", name, fl.queriedName)
	}
	elts := getQueueElements(ep.queue)
	if len(elts) != 0 {
		t.Errorf("Expected no elements in queue")
	}
}

func TestDelete(t *testing.T) {
	name := "foobar"
	_, ep := setup(name, nil)

	ep.OnDelete(&v1.PolicyNode{ObjectMeta: metav1.ObjectMeta{Name: name}})
	expect := []string{name}
	elts := getQueueElements(ep.queue)
	if !reflect.DeepEqual(expect, elts) {
		t.Errorf("Expected queue %s got %s", expect, elts)
	}
}
