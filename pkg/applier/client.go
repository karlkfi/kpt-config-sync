// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package applier

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/scheme"
	kubectlutil "k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// resourceClient is the client to update object in the API server.
type resourceClient struct {
	client     dynamic.Interface
	restMapper meta.RESTMapper
}

// newResourceClient returns a client to get and update an object.
func newResourceClient(d dynamic.Interface, mapper meta.RESTMapper) *resourceClient {
	return &resourceClient{
		client:     d,
		restMapper: mapper,
	}
}

// update updates an object using dynamic client
func (uc *resourceClient) update(ctx context.Context, meta object.ObjMetadata, obj *unstructured.Unstructured, options *metav1.UpdateOptions) error {
	r, err := uc.resourceInterface(meta)
	if err != nil {
		return err
	}
	if options == nil {
		options = &metav1.UpdateOptions{}
	}
	_, err = r.Update(ctx, obj, *options)
	return err
}

// get fetches the requested object into the input obj using dynamic client
func (uc *resourceClient) get(ctx context.Context, meta object.ObjMetadata) (*unstructured.Unstructured, error) {
	r, err := uc.resourceInterface(meta)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, meta.Name, metav1.GetOptions{})
}

func (uc *resourceClient) resourceInterface(meta object.ObjMetadata) (dynamic.ResourceInterface, error) {
	mapping, err := uc.restMapper.RESTMapping(meta.GroupKind)
	if err != nil {
		return nil, err
	}
	namespacedClient := uc.client.Resource(mapping.Resource).Namespace(meta.Namespace)
	return namespacedClient, nil
}

// updateLabelsAndAnnotations updates .metadata.labels and .metadata.annotations fields of obj to use
// the passed in labels and annotations.
// It also updates the last-applied-configuration annotations.
func updateLabelsAndAnnotations(obj *unstructured.Unstructured, labels, annotations map[string]string) error {
	u := getOriginalObj(obj)
	if u != nil {
		u.SetAnnotations(annotations)
		u.SetLabels(labels)
		// Since the annotation is updated, we also need to update the
		// last applied configuration annotation.
		err := kubectlutil.CreateOrUpdateAnnotation(true, u, scheme.DefaultJSONEncoder())
		obj.SetLabels(u.GetLabels())
		obj.SetAnnotations(u.GetAnnotations())
		return err
	}
	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	return nil
}

func getOriginalObj(obj *unstructured.Unstructured) *unstructured.Unstructured {
	annotations := obj.GetAnnotations()
	lastApplied, found := annotations[v1.LastAppliedConfigAnnotation]
	if !found {
		return nil
	}
	u := &unstructured.Unstructured{}
	err := json.Unmarshal([]byte(lastApplied), u)
	if err != nil {
		return nil
	}
	return u
}
