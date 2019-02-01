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

// Package decode is used for decoding serialized data in Nomos resources.
package decode

import (
	"fmt"

	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Decoder decodes GenericResources from PolicyNodes / ClusterPolicies to
// Unstructured structs.
type Decoder interface {
	// DecodeResources reads the bytes in the RawExtensions representing k8s
	// resources and returns a slice of all the resources grouped by their
	// respective GroupVersionKind.
	DecodeResources(genericResources ...nomosv1.GenericResources) (map[schema.GroupVersionKind][]*unstructured.Unstructured, error)
}

var _ Decoder = &GenericResourceDecoder{}

// GenericResourceDecoder implements Decoder.
type GenericResourceDecoder struct {
	decoder               runtime.Decoder
	unstructuredConverter runtime.UnstructuredConverter
}

// NewGenericResourceDecoder returns a new GenericResourceDecoder.
func NewGenericResourceDecoder(scheme *runtime.Scheme) *GenericResourceDecoder {
	return &GenericResourceDecoder{
		decoder:               serializer.NewCodecFactory(scheme).UniversalDeserializer(),
		unstructuredConverter: runtime.DefaultUnstructuredConverter,
	}
}

// DecodeResources implements Decoder.
func (d *GenericResourceDecoder) DecodeResources(genericResources ...nomosv1.GenericResources) (map[schema.GroupVersionKind][]*unstructured.Unstructured, error) {
	us := make(map[schema.GroupVersionKind][]*unstructured.Unstructured)
	for _, gr := range genericResources {
		for _, v := range gr.Versions {
			for _, genericObject := range v.Objects {
				gvk := schema.GroupVersionKind{Group: gr.Group, Version: v.Version, Kind: gr.Kind}
				var o runtime.Object
				o, _, err := d.decoder.Decode(genericObject.Raw, &gvk, o)
				if err != nil {
					return nil, errors.Wrapf(err, "could not decode runtime.Object from %q RawExtension bytes", gvk)
				}
				au, ok := o.(*unstructured.Unstructured)
				if !ok {
					m, err := d.unstructuredConverter.ToUnstructured(o)
					if err != nil {
						return nil, fmt.Errorf("could not treat GenericResource object %q as an unstructured.Unstructured", gvk)
					}
					au = &unstructured.Unstructured{Object: m}
					au.SetGroupVersionKind(gvk)
				}
				us[gvk] = append(us[gvk], au)
			}
		}
	}
	return us, nil
}
