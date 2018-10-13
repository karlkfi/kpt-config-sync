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

package transform

import (
	policyhierarchy "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/pkg/errors"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type clusterSet = map[string]bool

// ClusterSelectors contains all information needed to deliberate on whether a cluster
// matches a selector or not.
type ClusterSelectors struct {
	// Map from cluster name to the cluster definition.
	clusters map[string]clusterregistry.Cluster
	// Map from selector name to selector content.
	selectors map[string]policyhierarchy.ClusterSelector
	// Map of selector name to clusters matching that selector.
	selectorToClusters map[string]clusterSet
}

// HasCluster returns true if the supplied cluster name is a member of the clusters
// matching supplied selector.
func (stc *ClusterSelectors) HasCluster(selector, cluster string) bool {
	sel := stc.selectorToClusters[selector]
	if sel == nil {
		return false
	}
	return sel[cluster]
}

// addCluster adds the supplied cluster to the set of clusters matched by selector.
func (stc *ClusterSelectors) addCluster(selector, cluster string) {
	m := stc.selectorToClusters[selector]
	if m == nil {
		m = make(clusterSet)
		stc.selectorToClusters[selector] = m
	}
	m[cluster] = true
}

// Cluster returns the registry record of the cluster with the given name.
func (stc *ClusterSelectors) Cluster(name string) (clusterregistry.Cluster, bool) {
	e, ok := stc.clusters[name]
	return e, ok
}

//ClusterSelector returns the cluster selector definition for the selector with the given name.
func (stc *ClusterSelectors) ClusterSelector(name string) (policyhierarchy.ClusterSelector, bool) {
	e, ok := stc.selectors[name]
	return e, ok
}

// ForEachSelector runs f on each name and selector pair in this collection of
// selectors.
func (stc *ClusterSelectors) ForEachSelector(f func(name string, selector policyhierarchy.ClusterSelector)) {
	for name, selector := range stc.selectors {
		f(name, selector)
	}
}

// NewClusterSelectors returns a new cluster selection object.
func NewClusterSelectors(
	clusters []clusterregistry.Cluster,
	selectors []policyhierarchy.ClusterSelector,
) (*ClusterSelectors, error) {
	cc := &ClusterSelectors{
		clusters:           map[string]clusterregistry.Cluster{},
		selectors:          map[string]policyhierarchy.ClusterSelector{},
		selectorToClusters: map[string]clusterSet{},
	}
	// Populate the internal mappings:
	for _, cs := range selectors {
		name := cs.ObjectMeta.Name
		cc.selectors[name] = cs
		s, err := AsPopulatedSelector(&cs.Spec.Selector)
		if err != nil {
			return nil, errors.Wrapf(err, "while populating cluster selector: %q", name)
		}
		for _, cl := range clusters {
			cc.clusters[cl.ObjectMeta.Name] = cl
			if IsSelected(cl.Labels, s) {
				cc.addCluster(name, cl.ObjectMeta.Name)
			}
		}
	}
	return cc, nil
}
