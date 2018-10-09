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
