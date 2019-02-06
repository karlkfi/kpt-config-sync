/*
Copyright 2017 The Nomos Authors.
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

package meta

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectScope is the return type for APIInfo.GetScope
type ObjectScope string

const (
	// Cluster is an object scoped to the cluster
	Cluster = ObjectScope("cluster")
	// Namespace is an object scoped to namespace
	Namespace = ObjectScope("namespace")
	// NotFound is returned if the object does not exist in APIInfo
	NotFound = ObjectScope("notFound")
)

type apiInfoKey struct{}

// AddAPIInfo returns a copy of the Extension with the APIInfo set.
// The value is only accessible with GetAPIInfo.
func AddAPIInfo(r *ast.Root, apiInfo *APIInfo) {
	r.Data = r.Data.Add(apiInfoKey{}, apiInfo)
}

// GetAPIInfo gets the APIInfo from the Extension.
func GetAPIInfo(r *ast.Root) *APIInfo {
	return r.Data.Get(apiInfoKey{}).(*APIInfo)
}

// APIInfo allows for looking up the discovery metav1.APIResource information by group version kind
type APIInfo struct {
	resources map[schema.GroupVersionKind]metav1.APIResource
}

// NewAPIInfo returns a new APIInfo object
func NewAPIInfo(resourceLists []*metav1.APIResourceList) (*APIInfo, error) {
	resources := map[schema.GroupVersionKind]metav1.APIResource{}
	for _, resourceList := range resourceLists {
		groupVersion, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse discovery APIResourceList")
		}
		for _, resource := range resourceList.APIResources {
			resources[groupVersion.WithKind(resource.Kind)] = resource
		}
	}
	return &APIInfo{resources: resources}, nil
}

// GetScope returns the scope for the object.  If not found, NotFound will be returned.
func (a *APIInfo) GetScope(gvk schema.GroupVersionKind) ObjectScope {
	resource, found := a.resources[gvk]
	if !found {
		return NotFound
	}
	if resource.Namespaced {
		return Namespace
	}
	return Cluster
}

// Exists returns true if the GroupVersionKind is in the APIResources.
func (a *APIInfo) Exists(gvk schema.GroupVersionKind) bool {
	_, exists := a.resources[gvk]
	return exists
}

// AllowedVersions returns a list of the versions allowed for the passed Group/Kind
func (a *APIInfo) AllowedVersions(gk schema.GroupKind) []string {
	var results []string
	for gvk := range a.resources {
		if gvk.GroupKind() == gk {
			results = append(results, gvk.Version)
		}
	}
	return results
}
