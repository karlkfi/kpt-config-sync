/*
Copyright 2017 The Stolos Authors.
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

package labeling

import (
	"reflect"
	"testing"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestLabeling(t *testing.T) {
	m := meta_v1.ObjectMeta{}
	AddOriginLabel(&m)
	if m.Labels == nil {
		t.Errorf("Should have added map")
	}
	if m.Labels[OriginLabelKey] != OriginLabelValue {
		t.Errorf("Should have correct key/value in map")
	}
	if !HasOriginLabel(m) {
		t.Errorf("Should have found label in map")
	}

	selector := NewOriginSelector()
	if !selector.Matches(labels.Set(m.Labels)) {
		t.Errorf("Selector should match label")
	}

}

func TestAddOriginLabelToMap(t *testing.T) {
	for _, testcase := range []struct {
		Input map[string]string
	}{
		{nil},
		{map[string]string{}},
		{map[string]string{"foo": "value"}},
	} {
		labelMap := testcase.Input
		ret := AddOriginLabelToMap(labelMap)
		if labelMap != nil && !reflect.DeepEqual(ret, labelMap) {
			t.Errorf("Incorrect arg / return value state")
		}
		if ret[OriginLabelKey] != OriginLabelValue {
			t.Errorf("Did not add label to map")
		}
	}
}
