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

package labeling

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestLabeling(t *testing.T) {
	obj := &v1.Role{}
	testKey := "test"
	testValue := "true"
	testLabel := Label{key: testKey, value: testValue}

	if testLabel.IsSet(obj) {
		t.Errorf("Should not be set")
	}

	obj.ObjectMeta.Labels = testLabel.AddDeepCopy(obj.ObjectMeta.Labels)
	if !testLabel.IsSet(obj) {
		t.Errorf("Should be set")
	}

	if obj.ObjectMeta.Labels[testKey] != testValue {
		t.Errorf("Wrong key/value for label")
	}

	selector := testLabel.Selector()
	if !selector.Matches(labels.Set(testLabel.New())) {
		t.Errorf("Selector should match label")
	}

	m := map[string]string{}
	testLabel.AddTo(m)
	if m[testKey] != testValue {
		t.Errorf("Wrong key/value for label")
	}
}

func TestRemoveQuota(t *testing.T) {
	tests := []struct {
		in, out  map[string]string
		hasNomos bool
	}{
		{},
		{
			in:  map[string]string{},
			out: map[string]string{},
		},
		{
			in:  map[string]string{"foo": "bar"},
			out: map[string]string{"foo": "bar"},
		},
		{
			in: map[string]string{
				"foo":                           "bar",
				"configmanagement.gke.io/quota": "true",
			},
			out:      map[string]string{"foo": "bar"},
			hasNomos: true,
		},
	}
	for _, test := range tests {
		var c map[string]string
		if test.in != nil {
			c = map[string]string{}
			for k, v := range test.in {
				c[k] = v
			}
		}
		RemoveQuota(c)
		if !cmp.Equal(test.out, c) {
			t.Errorf("RemoveQuota(%+v)=%+v, want: %+v\ndiff:%v\n", test.in, c, test.out, cmp.Diff(test.out, c))
		}
		actualHas := HasQuota(test.in)
		if actualHas != test.hasNomos {
			t.Errorf("HasQuota(%+v)=%v, want: %v", test.in, actualHas, test.hasNomos)
		}
	}
}
