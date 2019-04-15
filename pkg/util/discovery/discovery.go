/*
Copyright 2017 The CSP Config Management Authors.
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

package discovery

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectScope is the return type for APIInfo.GetScope
type ObjectScope string

const (
	// ClusterScope is an object scoped to the cluster
	ClusterScope = ObjectScope("cluster")
	// NamespaceScope is an object scoped to namespace
	NamespaceScope = ObjectScope("namespace")
	// UnknownScope is returned if the object does not exist in APIInfo
	UnknownScope = ObjectScope("unknown")
)

type apiInfoKey struct{}

// AddAPIInfo returns a copy of the Extension with the APIInfo set.
// The value is only accessible with GetAPIInfo.
func AddAPIInfo(r *ast.Root, apiInfo *APIInfo) status.Error {
	var err status.Error
	r.Data, err = ast.Add(r.Data, apiInfoKey{}, apiInfo)
	return err
}

// GetAPIInfo gets the APIInfo from the Extension.
func GetAPIInfo(r *ast.Root) (*APIInfo, status.Error) {
	result, err := ast.Get(r.Data, apiInfoKey{})
	if err != nil {
		return nil, err
	}
	return result.(*APIInfo), nil
}

// APIInfo allows for looking up the discovery metav1.APIResource information by group version kind
type APIInfo struct {
	groupKindVersions map[schema.GroupKind][]string
	resources         map[schema.GroupVersionKind]metav1.APIResource
}

// NewAPIInfo returns a new APIInfo object
func NewAPIInfo(resourceLists []*metav1.APIResourceList) (*APIInfo, error) {
	resources := map[schema.GroupVersionKind]metav1.APIResource{}
	groupKindVersions := map[schema.GroupKind][]string{}
	for _, resourceList := range resourceLists {
		groupVersion, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse discovery APIResourceList")
		}
		for _, resource := range resourceList.APIResources {
			resources[groupVersion.WithKind(resource.Kind)] = resource
			gk := groupVersion.WithKind(resource.Kind).GroupKind()
			groupKindVersions[gk] = append(groupKindVersions[gk], groupVersion.Version)
		}
	}
	return &APIInfo{resources: resources, groupKindVersions: groupKindVersions}, nil
}

// AddCustomResources updates APIInfo with custom resource metadata from the provided CustomResourceDefinitions.
// It does not replace anything that already exists in APIInfo.
func (a *APIInfo) AddCustomResources(crds ...*v1beta1.CustomResourceDefinition) {
	for _, crd := range crds {
		crSpec := crd.Spec
		crNames := crSpec.Names
		group := crSpec.Group
		kind := crSpec.Names.Kind
		apiResourceTemplate := metav1.APIResource{
			Name:         crNames.Plural,
			SingularName: crNames.Singular,
			Namespaced:   crSpec.Scope == v1beta1.NamespaceScoped,
			Group:        group,
			Kind:         kind,
			ShortNames:   crNames.ShortNames,
			// TODO(sbochins): consider non-empty defaults for Categories and Verbs
		}

		gk := schema.GroupKind{Group: group, Kind: kind}
		setVersion := func(version string) {
			gvk := gk.WithVersion(version)
			if _, ok := a.resources[gvk]; ok {
				// We've already added information for this GroupVersionKind; don't add duplicate info.
				return
			}
			a.groupKindVersions[gk] = append(a.groupKindVersions[gk], version)

			apiResource := *apiResourceTemplate.DeepCopy()
			apiResource.Version = version
			a.resources[gk.WithVersion(version)] = apiResource
		}

		for _, v := range crSpec.Versions {
			if !v.Served {
				continue
			}
			setVersion(v.Name)
		}
		if version := crSpec.Version; version != "" {
			setVersion(version)
		}
	}
}

// GetScope returns the scope for the object.  If not found, UnknownScope will be returned.
func (a *APIInfo) GetScope(gvk schema.GroupVersionKind) ObjectScope {
	resource, found := a.resources[gvk]
	if !found {
		return UnknownScope
	}
	if resource.Namespaced {
		return NamespaceScope
	}
	return ClusterScope
}

// Exists returns true if the GroupVersionKind is in the APIResources.
func (a *APIInfo) Exists(gvk schema.GroupVersionKind) bool {
	_, exists := a.resources[gvk]
	return exists
}

// GroupKindExists returns true if the GroupKind is in the APIResources.
func (a *APIInfo) GroupKindExists(gk schema.GroupKind) bool {
	_, ok := a.groupKindVersions[gk]
	return ok
}

// AllowedVersions returns a list of the versions allowed for the passed Group/Kind.
func (a *APIInfo) AllowedVersions(gk schema.GroupKind) []string {
	return a.groupKindVersions[gk]
}

// GroupVersionKinds returns a set of GroupVersionKinds represented by the slice of Syncs with only
// Group and Kind specified.
func (a *APIInfo) GroupVersionKinds(syncs ...*v1.Sync) map[schema.GroupVersionKind]bool {
	allGvk := make(map[schema.GroupVersionKind]bool)
	for _, s := range syncs {
		gk := schema.GroupKind{Group: s.Spec.Group, Kind: s.Spec.Kind}
		for _, v := range a.AllowedVersions(gk) {
			allGvk[gk.WithVersion(v)] = true
		}
	}
	return allGvk
}
