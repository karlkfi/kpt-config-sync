package mocks

import (
	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/decode"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ decode.Decoder = &FakeDecoder{}

// FakeDecoder is a decoder used for testing.
type FakeDecoder struct {
	data map[schema.GroupVersionKind][]*unstructured.Unstructured
}

// NewFakeDecoder returns a new FakeDecoder.
func NewFakeDecoder(us []*unstructured.Unstructured) *FakeDecoder {
	m := make(map[schema.GroupVersionKind][]*unstructured.Unstructured)
	for _, u := range us {
		gvk := u.GroupVersionKind()
		m[gvk] = append(m[gvk], u)
	}

	return &FakeDecoder{data: m}
}

// UpdateScheme does nothing.
func (d *FakeDecoder) UpdateScheme(gvks map[schema.GroupVersionKind]bool) {
}

// DecodeResources returns fake data.
func (d *FakeDecoder) DecodeResources(_ ...nomosv1.GenericResources) (map[schema.GroupVersionKind][]*unstructured.Unstructured, error) {
	return d.data, nil
}
