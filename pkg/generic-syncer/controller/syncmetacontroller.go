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

package controller

import (
	"fmt"

	"github.com/google/nomos/clientgen/apis"
	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// RegisterGenericResources updates the scheme with resources declared in Syncs.
func RegisterGenericResources(cfg *rest.Config, scheme *runtime.Scheme,
	clientSet *apis.Clientset) ([]schema.GroupVersionKind, []schema.GroupVersionKind, error) {
	namespaceScoped, err := resourceScopes(cfg)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get resource scope information from discovery API")
	}
	syncInformer := clientSet.NomosV1alpha1().Syncs()
	// TODO(115420897): We need to continuously check for added/removed Syncs and restart the Manager with new controllers for the
	// generic resources we're syncing. We also need to update the scheme appropriately.
	syncs, err := syncInformer.List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not list Syncs")
	}
	namespace, cluster := extractGroupVersionKinds(namespaceScoped, syncs.Items...)

	addResourcesToScheme(scheme, namespace...)
	addResourcesToScheme(scheme, cluster...)
	return namespace, cluster, nil
}

func resourceScopes(cfg *rest.Config) (map[schema.GroupVersionKind]bool, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create discoveryclient")
	}
	groups, err := dc.ServerResources()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get api groups")
	}
	namespaceScoped := make(map[schema.GroupVersionKind]bool)
	for _, g := range groups {
		gv, grpErr := schema.ParseGroupVersion(g.GroupVersion)
		if grpErr != nil {
			// This shouldn't happen because we get these values from the server.
			return nil, fmt.Errorf("received invalid GroupVersion from server: %v", grpErr)
		}
		for _, apir := range g.APIResources {
			if apir.Namespaced {
				namespaceScoped[gv.WithKind(apir.Kind)] = true
			}
		}
	}
	return namespaceScoped, nil
}

// addsResourcesToScheme adds resources represented by GroupVersionKinds to the api scheme.
// This is needed by the kubebuilder APIs in order to generate informers/listers for GenericResources defined in
// PolicyNodes/ClusterPolicies.
func addResourcesToScheme(scheme *runtime.Scheme, gvks ...schema.GroupVersionKind) {
	for _, gvk := range gvks {
		if !scheme.Recognizes(gvk) {
			scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
			// TODO: see if we can avoid akwardly creating a list Kind.
			gvkList := schema.GroupVersionKind{
				Group:   gvk.Group,
				Version: gvk.Version,
				Kind:    gvk.Kind + "List",
			}
			scheme.AddKnownTypeWithName(gvkList, &unstructured.UnstructuredList{})
			metav1.AddToGroupVersion(scheme, gvk.GroupVersion())
		}
	}
}

func extractGroupVersionKinds(namespaceScoped map[schema.GroupVersionKind]bool,
	syncs ...nomosv1alpha1.Sync) (namespace []schema.GroupVersionKind, cluster []schema.GroupVersionKind) {
	for _, sync := range syncs {
		for _, g := range sync.Spec.Groups {
			k := g.Kinds
			for _, v := range k.Versions {
				gvk := schema.GroupVersionKind{
					Group:   g.Group,
					Version: v.Version,
					Kind:    k.Kind,
				}
				if namespaceScoped[gvk] {
					namespace = append(namespace, gvk)
				} else {
					cluster = append(cluster, gvk)
				}
			}
		}
	}
	return
}
