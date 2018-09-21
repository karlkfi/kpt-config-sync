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
	"github.com/google/nomos/clientgen/apis"
	"k8s.io/apimachinery/pkg/runtime/schema"

	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RegisterGenericResources updates the scheme with resources declared in Syncs.
func RegisterGenericResources(scheme *runtime.Scheme, clientSet *apis.Clientset) ([]schema.GroupVersionKind, error) {
	var gvks []schema.GroupVersionKind
	syncInformer := clientSet.NomosV1alpha1().Syncs()
	// TODO(115420897): We need to continuously check for added/removed Syncs and restart the Manager with new controllers for the
	// generic resources we're syncing. We also need to update the scheme appropriately.
	syncs, err := syncInformer.List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not list Syncs")
	}
	gvks = extractGroupVersionKinds(syncs.Items...)

	addResourcesToScheme(scheme, gvks)
	return gvks, nil
}

// addsResourcesToScheme adds resources represented by GroupVersionKinds to the api scheme.
// This is needed by the kubebuilder APIs in order to generate informers/listers for GenericResources defined in
// PolicyNodes/ClusterPolicies.
func addResourcesToScheme(scheme *runtime.Scheme, gvks []schema.GroupVersionKind) {
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

func extractGroupVersionKinds(syncDeclarations ...nomosv1alpha1.Sync) (gvks []schema.GroupVersionKind) {
	for _, syncDeclaration := range syncDeclarations {
		for _, group := range syncDeclaration.Spec.Groups {
			kind := group.Kinds
			for _, version := range kind.Versions {
				gvks = append(gvks, schema.GroupVersionKind{
					Group:   group.Group,
					Version: version.Version,
					Kind:    kind.Kind,
				})
			}
		}
	}
	return
}
