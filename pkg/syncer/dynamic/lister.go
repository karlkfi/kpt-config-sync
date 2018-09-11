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

package dynamic

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// Lister lists arbitrary resources specified by the Informer that created it.
type Lister interface {
	List(selector labels.Selector) (ret []*unstructured.Unstructured, err error)
	ListNamespace(namespace string, selector labels.Selector) (ret []*unstructured.Unstructured, err error)
	Get(namespace, name string) (*unstructured.Unstructured, error)
}

type lister struct {
	indexer       cache.Indexer
	groupResource schema.GroupResource
}

func newLister(indexer cache.Indexer, resource schema.GroupResource) Lister {
	return &lister{
		indexer:       indexer,
		groupResource: resource,
	}
}

// List lists resources in the indexer.
func (l *lister) List(selector labels.Selector) (ret []*unstructured.Unstructured, err error) {
	err = cache.ListAll(l.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*unstructured.Unstructured))
	})
	return ret, err
}

// ListNamespace lists resources in the indexer with the namespace.
func (l *lister) ListNamespace(namespace string, selector labels.Selector) (ret []*unstructured.Unstructured, err error) {
	err = cache.ListAllByNamespace(l.indexer, namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*unstructured.Unstructured))
	})
	return ret, err
}

// Get gets a resources in the indexer with the name.
func (l *lister) Get(namespace, name string) (*unstructured.Unstructured, error) {
	key := name
	if namespace != "" {
		key = fmt.Sprintf("%s/%s", namespace, name)
	}
	obj, exists, err := l.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(l.groupResource, name)
	}
	return obj.(*unstructured.Unstructured), nil
}
