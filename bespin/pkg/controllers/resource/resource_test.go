/*
Copyright 2019 Google LLC.

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

package resource

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	bespinv1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	syncToken1 = "synctoken1"
	syncToken2 = "synctoken2"
)

func TestAnnotateSyncDetails(t *testing.T) {
	oldSyncTime := "2019-01-01T00:20:24Z"
	now := metav1.Now().Format(time.RFC3339)
	tests := []struct {
		name string
		obj  GenericObject
		want GenericObject
	}{
		// After annotateSyncDetails the syncToken should be the same as importToken,
		// and syncTime should not be empty.
		{
			name: "Sync token is nil, sync time is nil",
			obj: &bespinv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
					},
				},
			},
			want: &bespinv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
						syncTokenKey:   syncToken1,
						syncTimeKey:    now,
					},
				},
			},
		},
		{
			name: "Sync token is different from import token, sync time is nil",
			obj: &bespinv1.Folder{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
						syncTokenKey:   syncToken2,
					},
				},
			},
			want: &bespinv1.Folder{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
						syncTokenKey:   syncToken1,
						syncTimeKey:    now,
					},
				},
			},
		},
		{
			name: "Sync token is different from import token, sync time is NOT nil",
			obj: &bespinv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
						syncTokenKey:   syncToken2,
						syncTimeKey:    oldSyncTime,
					},
				},
			},
			want: &bespinv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
						syncTokenKey:   syncToken1,
						syncTimeKey:    now,
					},
				},
			},
		},
		{
			name: "Sync token is the same as import token, sync time NOT nil, don't update the sync time",
			obj: &bespinv1.OrganizationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
						syncTokenKey:   syncToken1,
						syncTimeKey:    oldSyncTime,
					},
				},
			},
			want: &bespinv1.OrganizationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ImportTokenKey: syncToken1,
						syncTokenKey:   syncToken1,
						syncTimeKey:    oldSyncTime,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			annotateOKSync(tc.obj)
			gotAnnotations := tc.obj.GetAnnotations()
			gotSyncTime, err := time.Parse(time.RFC3339, gotAnnotations[syncTimeKey])
			if err != nil {
				t.Fatalf("got invalid syncTime %v: %v", gotAnnotations[syncTimeKey], err)
			}
			wantAnnotations := tc.want.GetAnnotations()
			wantSyncTime, err := time.Parse(time.RFC3339, wantAnnotations[syncTimeKey])
			if err != nil {
				t.Fatalf("got invalid syncTime %v: %v", wantAnnotations[syncTimeKey], err)
			}
			if gotSyncTime.Sub(wantSyncTime).Seconds() > 5.0 {
				t.Fatalf("sync time doesn't match, got: %v, want: %v", gotSyncTime, wantSyncTime)
			}
			gotAnnotations[syncTimeKey] = wantAnnotations[syncTimeKey]
			if diff := cmp.Diff(tc.obj, tc.want); diff != "" {
				t.Errorf("got diff:\n%v", diff)
			}
		})
	}
}
