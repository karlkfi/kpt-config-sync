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
package reserved

import (
	"testing"

	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReservedNamespaces(t *testing.T) {
	testCases := []struct {
		name      string
		configMap *v1.ConfigMap
		wantErr   error
	}{
		{
			name: "valid",
			configMap: &v1.ConfigMap{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: policyhierarchyv1.ReservedNamespacesConfigMapName,
				},
				Data: map[string]string{
					"eng":      "reserved",
					"backend":  "reserved",
					"frontend": "reserved",
				},
			},
		},
		{
			name: "invalid attribute",
			configMap: &v1.ConfigMap{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: policyhierarchyv1.ReservedNamespacesConfigMapName,
				},
				Data: map[string]string{
					"eng":      "reserved",
					"backend":  "reserved",
					"frontend": "unreserved",
				},
			},
			wantErr: errors.Errorf("the reserved namespace, backend attribute is invalid: managed"),
		},
		{
			name: "invalid namespace name",
			configMap: &v1.ConfigMap{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: policyhierarchyv1.ReservedNamespacesConfigMapName,
				},
				Data: map[string]string{
					"eng":       "reserved",
					"backend":   "reserved",
					"frontend@": "reserved",
				},
			},
			wantErr: errors.Errorf("the reserved namespace name, frontend@ is invalid: "),
		},
		{
			name: "invalid configmap name",
			configMap: &v1.ConfigMap{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: "some-incorrect-name",
				},
				Data: map[string]string{
					"eng":       "reserved",
					"backend":   "reserved",
					"frontend@": "reserved",
				},
			},
			wantErr: errors.Errorf("the reserved namespace configmap name is invalid"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := From(tt.configMap)
			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) {
				t.Errorf("Unexpected error when validating:\n%v\nwant: %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsReservedNamespace(t *testing.T) {
	configMap := &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: policyhierarchyv1.ReservedNamespacesConfigMapName,
		},
		Data: map[string]string{
			"eng":      "reserved",
			"backend":  "reserved",
			"frontend": "reserved",
		},
	}

	testCases := []struct {
		name      string
		namespace string
		configMap *v1.ConfigMap
		want      bool
	}{
		{
			name:      "reserved namespace",
			namespace: "eng",
			configMap: configMap,
			want:      true,
		},
		{
			name:      "non-reserved namespace",
			namespace: "foobar",
			configMap: configMap,
			want:      false,
		},
		{
			name:      "nil configMap still generates namespaces",
			namespace: "eng",
			configMap: nil,
			want:      false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ns, err := From(tt.configMap)
			if err != nil {
				t.Errorf("could not generate stub Namespaces: %v", err)
			}
			if got := ns.IsReserved(tt.namespace); got != tt.want {
				t.Errorf("want unamanged: %t, got %t", tt.want, got)
			}
		})
	}
}
