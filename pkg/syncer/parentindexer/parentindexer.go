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

// Package parentindexer adds an indexer for policy nodes that provides parent to child
// mapping for indexed lookup.
package parentindexer

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"k8s.io/client-go/tools/cache"
)

// Name is the name of the indexer
const parentIndex = "PolicyNode-Parent"

// policyNodeIndexParent returns the index key (parnent) for the given policy node.
func policyNodeIndexParent(obj interface{}) ([]string, error) {
	policyNode := obj.(*v1.PolicyNode)
	return []string{policyNode.Spec.Parent}, nil
}

// Indexer returns the indexer to index by parent node.
func Indexer() cache.Indexers {
	return cache.Indexers{
		parentIndex: policyNodeIndexParent,
	}
}

// GetChildNodes will return the policy nodes that are children of parent.
func GetChildNodes(informer cache.SharedIndexInformer, parent string) ([]*v1.PolicyNode, error) {
	objs, err := informer.GetIndexer().ByIndex(parentIndex, parent)
	if err != nil {
		return nil, err
	}

	policyNodes := make([]*v1.PolicyNode, len(objs))
	for idx, obj := range objs {
		policyNodes[idx] = obj.(*v1.PolicyNode)
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
