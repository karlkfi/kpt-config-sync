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

// This file was automatically generated by lister-gen

package v1

import (
	v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// StolosResourceQuotaLister helps list StolosResourceQuotas.
type StolosResourceQuotaLister interface {
	// List lists all StolosResourceQuotas in the indexer.
	List(selector labels.Selector) (ret []*v1.StolosResourceQuota, err error)
	// StolosResourceQuotas returns an object that can list and get StolosResourceQuotas.
	StolosResourceQuotas(namespace string) StolosResourceQuotaNamespaceLister
	StolosResourceQuotaListerExpansion
}

// stolosResourceQuotaLister implements the StolosResourceQuotaLister interface.
type stolosResourceQuotaLister struct {
	indexer cache.Indexer
}

// NewStolosResourceQuotaLister returns a new StolosResourceQuotaLister.
func NewStolosResourceQuotaLister(indexer cache.Indexer) StolosResourceQuotaLister {
	return &stolosResourceQuotaLister{indexer: indexer}
}

// List lists all StolosResourceQuotas in the indexer.
func (s *stolosResourceQuotaLister) List(selector labels.Selector) (ret []*v1.StolosResourceQuota, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.StolosResourceQuota))
	})
	return ret, err
}

// StolosResourceQuotas returns an object that can list and get StolosResourceQuotas.
func (s *stolosResourceQuotaLister) StolosResourceQuotas(namespace string) StolosResourceQuotaNamespaceLister {
	return stolosResourceQuotaNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// StolosResourceQuotaNamespaceLister helps list and get StolosResourceQuotas.
type StolosResourceQuotaNamespaceLister interface {
	// List lists all StolosResourceQuotas in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.StolosResourceQuota, err error)
	// Get retrieves the StolosResourceQuota from the indexer for a given namespace and name.
	Get(name string) (*v1.StolosResourceQuota, error)
	StolosResourceQuotaNamespaceListerExpansion
}

// stolosResourceQuotaNamespaceLister implements the StolosResourceQuotaNamespaceLister
// interface.
type stolosResourceQuotaNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all StolosResourceQuotas in the indexer for a given namespace.
func (s stolosResourceQuotaNamespaceLister) List(selector labels.Selector) (ret []*v1.StolosResourceQuota, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.StolosResourceQuota))
	})
	return ret, err
}

// Get retrieves the StolosResourceQuota from the indexer for a given namespace and name.
func (s stolosResourceQuotaNamespaceLister) Get(name string) (*v1.StolosResourceQuota, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("stolosresourcequota"), name)
	}
	return obj.(*v1.StolosResourceQuota), nil
}
