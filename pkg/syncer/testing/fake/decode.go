package fake

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/decode"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ decode.Decoder = &Decoder{}

// Decoder is a decoder used for testing.
type Decoder struct {
	data map[schema.GroupVersionKind][]*unstructured.Unstructured
}

// NewDecoder returns a new Decoder.
func NewDecoder(us []*unstructured.Unstructured) *Decoder {
	m := make(map[schema.GroupVersionKind][]*unstructured.Unstructured)
	for _, u := range us {
		gvk := u.GroupVersionKind()
		m[gvk] = append(m[gvk], u)
	}

	return &Decoder{data: m}
}

// UpdateScheme does nothing.
func (d *Decoder) UpdateScheme(gvks map[schema.GroupVersionKind]bool) {
}

// DecodeResources returns fake data.
func (d *Decoder) DecodeResources(_ []v1.GenericResources) (map[schema.GroupVersionKind][]*unstructured.Unstructured, error) {
	return d.data, nil
}
