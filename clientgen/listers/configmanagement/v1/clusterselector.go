// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ClusterSelectorLister helps list ClusterSelectors.
type ClusterSelectorLister interface {
	// List lists all ClusterSelectors in the indexer.
	List(selector labels.Selector) (ret []*v1.ClusterSelector, err error)
	// Get retrieves the ClusterSelector from the index for a given name.
	Get(name string) (*v1.ClusterSelector, error)
	ClusterSelectorListerExpansion
}

// clusterSelectorLister implements the ClusterSelectorLister interface.
type clusterSelectorLister struct {
	indexer cache.Indexer
}

// NewClusterSelectorLister returns a new ClusterSelectorLister.
func NewClusterSelectorLister(indexer cache.Indexer) ClusterSelectorLister {
	return &clusterSelectorLister{indexer: indexer}
}

// List lists all ClusterSelectors in the indexer.
func (s *clusterSelectorLister) List(selector labels.Selector) (ret []*v1.ClusterSelector, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ClusterSelector))
	})
	return ret, err
}

// Get retrieves the ClusterSelector from the index for a given name.
func (s *clusterSelectorLister) Get(name string) (*v1.ClusterSelector, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("clusterselector"), name)
	}
	return obj.(*v1.ClusterSelector), nil
}
