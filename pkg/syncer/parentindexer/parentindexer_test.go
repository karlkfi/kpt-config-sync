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

package parentindexer

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/google/nomos/clientgen/apis/fake"
	"github.com/google/nomos/clientgen/informer"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/nomos/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

func newPolicyNode(name, parent string) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       reflect.TypeOf(policyhierarchy_v1.PolicyNode{}).Name(),
			APIVersion: policyhierarchy_v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Parent: parent,
		},
	}
}

type Spec struct {
	name   string
	parent string
}

func (s *Spec) node() *policyhierarchy_v1.PolicyNode {
	return newPolicyNode(s.name, s.parent)
}

type Specs []Spec

func (s Specs) nodes() (nodes []*policyhierarchy_v1.PolicyNode) {
	for _, spec := range s {
		nodes = append(nodes, spec.node())
	}
	return nodes
}

func (s Specs) children(node string) (nodes []string) {
	for _, spec := range s {
		if spec.parent == node {
			nodes = append(nodes, spec.name)
		}
	}
	return nodes
}

func expectChildren(t *testing.T, informer cache.SharedIndexInformer, node string, expect []string) {
	children, err := GetChildren(informer, node)
	if err != nil {
		t.Errorf("Got error %s", err)
	}

	// reflect.DeepEqual treats nil slice and zero sized slice as not equal.
	if len(children) == 0 && len(expect) == 0 {
		return
	}

	sort.StringSlice(expect).Sort()
	sort.StringSlice(children).Sort()
	if !reflect.DeepEqual(expect, children) {
		t.Errorf("node %s expected children %v got %v", node, expect, children)
	}
}

func TestIndexer(t *testing.T) {
	specs := Specs{
		{name: "root", parent: ""},
		{name: "no-children", parent: "root"},
		{name: "has-children", parent: "root"},
		{name: "child1", parent: "has-children"},
		{name: "child2", parent: "has-children"},
		{name: "child3", parent: "has-children"},
	}

	var objs []runtime.Object
	for _, node := range specs.nodes() {
		objs = append(objs, node)
	}
	client := fake.NewSimpleClientset(objs...)
	informerFactory := informer.NewSharedInformerFactory(client, 2*time.Minute)
	informer := informerFactory.Nomos().V1().PolicyNodes().Informer()
	if err := informer.AddIndexers(Indexer()); err != nil {
		t.Errorf("unexpected error %s", err)
	}

	informerFactory.Start(nil)
	for k, v := range informerFactory.WaitForCacheSync(nil) {
		if !v {
			t.Errorf("Failed to sync %v", k)
		}
	}

	// Self-test
	expectChildren(t, informer, "", []string{"root"})
	expectChildren(t, informer, "has-children", []string{"child1", "child2", "child3"})

	for _, spec := range specs {
		expectChildren(t, informer, spec.name, specs.children(spec.name))
	}
}
