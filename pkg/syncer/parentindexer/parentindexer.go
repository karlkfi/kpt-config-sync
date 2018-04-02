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

// Package policynodeindexer adds an indexer for policy nodes that provides parent to child
// mapping for indexed lookup.
package policynodeindexer

import (
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/cache"
)

// Name is the name of the indexer
const parentIndex = "PolicyNode-Parent"

var indexer = cache.Indexers{
	parentIndex: policyNodeIndexParent,
}

// ParentIndexer indexes a policy node by parent.
type ParentIndexer struct {
	informerProvider informers.InformerProvider
}

var _ informers.InformerProvider = &ParentIndexer{}

// Informer implements informers.InformerProvider
func (s *ParentIndexer) Informer() cache.SharedIndexInformer {
	informer := s.informerProvider.Informer()
	if err := informer.AddIndexers(indexer); err != nil {
		// This will only fail if the informer is already running, or if someone has used the same
		// indexer key as us.
		panic(errors.Errorf("Failed to add policy node parent indexer"))
	}
	return informer
}

// policyNodeIndexParent returns the index key (parnent) for the given policy node.
func policyNodeIndexParent(obj interface{}) ([]string, error) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	return []string{policyNode.Spec.Parent}, nil
}

// Wrap will create a new informer provider that adds another indexer to the previous provider's
// informer.
func Wrap(informerProvider informers.InformerProvider) *ParentIndexer {
	return &ParentIndexer{informerProvider: informerProvider}
}

// GetChildNodes will return the policy nodes that are children of parent.
func GetChildNodes(informer cache.SharedIndexInformer, parent string) ([]*policyhierarchy_v1.PolicyNode, error) {
	objs, err := informer.GetIndexer().ByIndex(parentIndex, parent)
	if err != nil {
		return nil, err
	}

	policyNodes := make([]*policyhierarchy_v1.PolicyNode, len(objs))
	for idx, obj := range objs {
		policyNodes[idx] = obj.(*policyhierarchy_v1.PolicyNode)
	}
	return policyNodes, nil
}

// GetChildren will return the names of the children for parent.
func GetChildren(informer cache.SharedIndexInformer, parent string) ([]string, error) {
	nodes, err := GetChildNodes(informer, parent)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(nodes))
	for idx, node := range nodes {
		names[idx] = node.Name
	}
	return names, nil
}
