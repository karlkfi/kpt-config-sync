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

package testing

import (
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
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

// DecodeResources returns fake data.
func (d *FakeDecoder) DecodeResources(_ ...nomosv1.GenericResources) (map[schema.GroupVersionKind][]*unstructured.Unstructured, error) {
	return d.data, nil
}
