/*
Copyright 2017 The Nomos Authors.
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
// Reviewed by sunilarora
package actions

import (
	"reflect"
	"testing"

	"github.com/google/nomos/pkg/client/action"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetNamespaceLabelsFunc(t *testing.T) {
	var testcases = []struct {
		name           string
		nsLabels       map[string]string
		updateLabels   map[string]string
		expectLabels   map[string]string
		expectNoUpdate bool
	}{
		{
			name:         "Add to labels",
			nsLabels:     map[string]string{"foo-1": "bar-1"},
			updateLabels: map[string]string{"foo-2": "bar-2"},
			expectLabels: map[string]string{"foo-1": "bar-1", "foo-2": "bar-2"},
		},
		{
			name:         "Update existing label",
			nsLabels:     map[string]string{"foo-1": "bar-1"},
			updateLabels: map[string]string{"foo-1": "new-value"},
			expectLabels: map[string]string{"foo-1": "new-value"},
		},
		{
			name:         "Update one label",
			nsLabels:     map[string]string{"foo-1": "bar-1", "foo-2": "bar-2"},
			updateLabels: map[string]string{"foo-1": "new-value"},
			expectLabels: map[string]string{"foo-1": "new-value", "foo-2": "bar-2"},
		},
		{
			name:         "Update multiple labels",
			nsLabels:     map[string]string{"foo-1": "bar-1", "foo-2": "bar-2"},
			updateLabels: map[string]string{"foo-1": "new-value", "foo-2": "new-value-2"},
			expectLabels: map[string]string{"foo-1": "new-value", "foo-2": "new-value-2"},
		},
		{
			name:           "No update needed emtpy",
			nsLabels:       map[string]string{},
			updateLabels:   map[string]string{},
			expectNoUpdate: true,
		},
		{
			name:           "No update needed one elt",
			nsLabels:       map[string]string{"foo-1": "bar-1"},
			updateLabels:   map[string]string{"foo-1": "bar-1"},
			expectNoUpdate: true,
		},
		{
			name:           "No update needed two elts",
			nsLabels:       map[string]string{"foo-1": "bar-1", "foo-2": "bar-2"},
			updateLabels:   map[string]string{"foo-1": "bar-1", "foo-2": "bar-2"},
			expectNoUpdate: true,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: tt.nsLabels,
				},
			}

			f := SetNamespaceLabelsFunc(tt.updateLabels)
			obj, err := f(ns)
			if tt.expectNoUpdate {
				if err == nil {
					t.Errorf("Expected error")
				}
				if !action.IsNoUpdateNeeded(err) {
					t.Errorf("Expected no update needed error, got %s", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error %s", err)
			}

			nsObj := obj.(*corev1.Namespace)
			if !reflect.DeepEqual(nsObj.Labels, tt.expectLabels) {
				t.Errorf("new labels %v differ from expected %v", nsObj.Labels, tt.expectLabels)
			}
		})
	}
}
