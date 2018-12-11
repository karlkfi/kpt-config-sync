/*
Copyright 2018 Google LLC.

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

package apiobject

import (
	"reflect"
	"testing"

	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnnotate(t *testing.T) {
	tests := []struct {
		name string
		rs   Resource
		aKey string
		aVal string
		want map[string]string
	}{
		{
			name: "Annotate Project with Parent Organization ID",
			rs: &bespinv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1": "value1",
					},
				},
			},
			aKey: bespinv1.ParentOrganizationIDKey,
			aVal: "1234567",
			want: map[string]string{
				"key1":                           "value1",
				bespinv1.ParentOrganizationIDKey: "1234567",
			},
		},
		{
			name: "Annotate Project with Parent Folder ID",
			rs: &bespinv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1": "value1",
					},
				},
			},
			aKey: bespinv1.ParentFolderIDKey,
			aVal: "1234567",
			want: map[string]string{
				"key1":                     "value1",
				bespinv1.ParentFolderIDKey: "1234567",
			},
		},
		{
			name: "Annotate Folder with Parent Organization ID",
			rs: &bespinv1.Folder{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1": "value1",
					},
				},
			},
			aKey: bespinv1.ParentOrganizationIDKey,
			aVal: "1234567",
			want: map[string]string{
				"key1":                           "value1",
				bespinv1.ParentOrganizationIDKey: "1234567",
			},
		},
		{
			name: "Annotate Folder with Parent Folder ID",
			rs: &bespinv1.Folder{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1": "value1",
					},
				},
			},
			aKey: bespinv1.ParentFolderIDKey,
			aVal: "1234567",
			want: map[string]string{
				"key1":                     "value1",
				bespinv1.ParentFolderIDKey: "1234567",
			},
		},
		{
			name: "Annotation key already exists",
			rs: &bespinv1.Folder{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1":                     "value1",
						bespinv1.ParentFolderIDKey: "7654321",
					},
				},
			},
			aKey: bespinv1.ParentFolderIDKey,
			aVal: "1234567",
			want: map[string]string{
				"key1":                     "value1",
				bespinv1.ParentFolderIDKey: "1234567",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			annotate(tc.rs, tc.aKey, tc.aVal)
			got := tc.rs.GetAnnotations()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("annotate() got %+v; want %+v", got, tc.want)
			}
		})
	}
}
