// Package decode is used for decoding serialized data in Nomos resources.
package decode

import (
	"fmt"

	"github.com/google/nomos/pkg/syncer/scheme"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Decoder decodes GenericResources from NamespaceConfigs / ClusterConfigs to
// Unstructured structs.
type Decoder interface {
	// DecodeResources reads the bytes in the RawExtensions representing k8s
	// resources and returns a slice of all the resources grouped by their
	// respective GroupVersionKind.
	DecodeResources(genericResources []v1.GenericResources) (map[schema.GroupVersionKind][]*unstructured.Unstructured, error)
	// UpdateScheme updates the scheme of the underlying decoder, so it can decode the given GroupVersionKinds.
	UpdateScheme(gvks map[schema.GroupVersionKind]bool)
}

var _ Decoder = &genericResourceDecoder{}

// genericResourceDecoder implements Decoder.
type genericResourceDecoder struct {
	scheme                *runtime.Scheme
	decoder               runtime.Decoder
	unstructuredConverter runtime.UnstructuredConverter
}

// NewGenericResourceDecoder returns a new genericResourceDecoder.
func NewGenericResourceDecoder(scheme *runtime.Scheme) Decoder {
	return &genericResourceDecoder{
		scheme:                scheme,
		decoder:               serializer.NewCodecFactory(scheme).UniversalDeserializer(),
		unstructuredConverter: runtime.DefaultUnstructuredConverter,
	}
}

// UpdateScheme implements Decoder.
func (d *genericResourceDecoder) UpdateScheme(gvks map[schema.GroupVersionKind]bool) {
	scheme.AddToSchemeAsUnstructured(d.scheme, gvks)
	d.decoder = serializer.NewCodecFactory(d.scheme).UniversalDeserializer()
}

// DecodeResources implements Decoder.
func (d *genericResourceDecoder) DecodeResources(genericResources []v1.GenericResources) (map[schema.GroupVersionKind][]*unstructured.Unstructured, error) {
	us := make(map[schema.GroupVersionKind][]*unstructured.Unstructured)
	for _, gr := range genericResources {
		for _, v := range gr.Versions {
			for _, genericObject := range v.Objects {
				gvk := schema.GroupVersionKind{Group: gr.Group, Version: v.Version, Kind: gr.Kind}
				o := genericObject.Object
				if o == nil {
					u := &unstructured.Unstructured{}
					var err error
					o, _, err = d.decoder.Decode(genericObject.Raw, &gvk, u)
					if err != nil {
						return nil, errors.Wrapf(err, "could not decode runtime.Object from %q RawExtension bytes", gvk)
					}
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
