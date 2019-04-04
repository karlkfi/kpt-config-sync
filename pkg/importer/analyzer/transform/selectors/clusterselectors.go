/*
Copyright 2018 The CSP Config Management Authors.

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

package selectors

import (
	"reflect"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/status"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ClusterSelectors contains all information needed to deliberate on whether a cluster
// matches a selector or not.
type ClusterSelectors struct {
	// The cluster registry object corresponding to this cluster.
	cluster clusterregistry.Cluster
	// A set of selectors matching this cluster
	selectors map[string]v1.ClusterSelector
	// The name of the current cluster, if a name is known.
	clusterName string
}

// Equal returns true if c and other are exactly equal.
func (stc ClusterSelectors) Equal(other ClusterSelectors) bool {
	return reflect.DeepEqual(stc.cluster, other.cluster) &&
		reflect.DeepEqual(stc.selectors, other.selectors) &&
		reflect.DeepEqual(stc.clusterName, other.clusterName)
}

type clusterSelectorKeyType struct{}

var csKey = clusterSelectorKeyType{}

// SetClusterSelector extends root with the cluster selector.  Use
// GetClusterSelectors() to get it back.
func SetClusterSelector(stc *ClusterSelectors, root *ast.Root) *ast.Root {
	root.Data = root.Data.Add(csKey, stc)
	return root
}

// GetClusterSelectors gets the cluster selectors object from the root.  Panics
// if not found.
func GetClusterSelectors(root *ast.Root) *ClusterSelectors {
	return root.Data.Get(csKey).(*ClusterSelectors)
}

// ForEachSelector runs f on each name and selector pair in this collection of
// selectors.
func (stc *ClusterSelectors) ForEachSelector(f func(name string, selector v1.ClusterSelector)) {
	for name, selector := range stc.selectors {
		f(name, selector)
	}
}

// NewClusterSelectors returns a new cluster selection object.
func NewClusterSelectors(
	clusters []clusterregistry.Cluster,
	selectors []v1.ClusterSelector,
	clusterName string,
) (*ClusterSelectors, status.Error) {
	glog.V(2).Infof("Cluster name: %q", clusterName)
	cc := &ClusterSelectors{
		selectors:   make(map[string]v1.ClusterSelector),
		clusterName: clusterName,
	}
	for _, cl := range clusters {
		if clusterName == cl.ObjectMeta.Name {
			cc.cluster = cl
			break
		}
	}
	for _, cs := range selectors {
		name := cs.ObjectMeta.Name
		s, err := AsPopulatedSelector(&cs.Spec.Selector)
		if err != nil {
			return nil, vet.InvalidSelectorError{Name: name, Cause: err}
		}
		if IsSelected(cc.cluster.ObjectMeta.Labels, s) {
			cc.selectors[name] = cs
		}
	}
	return cc, nil
}

// ClusterName returns the current cluster's name if known, or "" otherwise.
func (stc *ClusterSelectors) ClusterName() string {
	return stc.clusterName
}

// Matches checks if the supplied annotated object matches the selector.
func (stc *ClusterSelectors) Matches(o object.Annotated) bool {
	a := o.GetAnnotations()
	if glog.V(7) {
		glog.Infof("annotations: %+v", a)
	}
	selector, ok := a[v1.ClusterSelectorAnnotationKey]
	if !ok {
		// An object that is not annotated always matches.
		return true
	}
	clusterSelector, ok := stc.selectors[selector]
	if !ok {
		// No selector that matches this cluster also matches the selector name.
		return false
	}
	if glog.V(6) {
		glog.Infof("clusterSelector: %+v, clusterName: %+v", clusterSelector, stc.clusterName)
	}
	return true
}
