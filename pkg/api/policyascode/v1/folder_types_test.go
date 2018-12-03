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

package v1

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageFolder(t *testing.T) {
	key := types.NamespacedName{Name: "foo"}
	created := &Folder{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec: FolderSpec{
			ParentReference: ParentReference{
				Kind: OrganizationKind,
				Name: "bar",
			},
			DisplayName:   "spec-bar",
			ID:            1,
			ImportDetails: fakeImportDetails,
		},
		Status: FolderStatus{
			SyncDetails: fakeSyncDetails,
		},
	}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &Folder{}
	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(created))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.HaveOccurred())
}

func TestFolderGetTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		f       *Folder
		want    string
		wantErr error
	}{
		{
			name: "Folder with Organization as parent",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: FolderSpec{
					ParentReference: ParentReference{
						Kind: "Organization",
						Name: "organizations/1234567",
					},
					DisplayName:   "spec-bar",
					ImportDetails: fakeImportDetails,
				},
				Status: FolderStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want: `resource "google_folder" "bespin_folder" {
display_name = "spec-bar"
parent = "organizations/1234567"
}`,
			wantErr: nil,
		},
		{
			name: "Folder with Folder as parent",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: FolderSpec{
					ParentReference: ParentReference{
						Kind: "Folder",
						Name: "folders/1234567",
					},
					DisplayName:   "spec-bar",
					ImportDetails: fakeImportDetails,
				},
				Status: FolderStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want: `resource "google_folder" "bespin_folder" {
display_name = "spec-bar"
parent = "folders/1234567"
}`,
			wantErr: nil,
		},
		{
			name: "Project with invalid ParentReference",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: FolderSpec{
					ParentReference: ParentReference{
						Kind: "Invalid",
						Name: "bar",
					},
					DisplayName:   "spec-bar",
					ImportDetails: fakeImportDetails,
				},
				Status: FolderStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want:    "",
			wantErr: fmt.Errorf("invalid parent reference kind: Invalid"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.f.GetTFResourceConfig()
			switch {
			case tc.wantErr == nil && err != nil:
				t.Errorf("GetTFResourceConfig() got err %+v; want nil", err)
			case tc.wantErr != nil && err == nil:
				t.Errorf("GetTFResourceConfig() got nil; want err %+v", tc.wantErr)
			case got != tc.want:
				t.Errorf("GetTFResourceConfig() got \n%s\n want \n%s", got, tc.want)
			}
		})

	}
}

func TestFolderGetID(t *testing.T) {
	tests := []struct {
		name string
		f    *Folder
		want string
	}{
		{
			name: "Folder with Organization as parent",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: FolderSpec{
					ParentReference: ParentReference{
						Kind: "Organization",
						Name: "organizations/1234567",
					},
					DisplayName:   "spec-bar",
					ID:            7654321,
					ImportDetails: fakeImportDetails,
				},
				Status: FolderStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want: "7654321",
		},
		{
			name: "Folder with Folder as parent",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: FolderSpec{
					ParentReference: ParentReference{
						Kind: "Folder",
						Name: "folders/1234567",
					},
					DisplayName:   "spec-bar",
					ID:            9876543,
					ImportDetails: fakeImportDetails,
				},
				Status: FolderStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want: "9876543",
		},
		{
			name: "Project with no ID",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: FolderSpec{
					ParentReference: ParentReference{
						Kind: "Invalid",
						Name: "bar",
					},
					DisplayName:   "spec-bar",
					ImportDetails: fakeImportDetails,
				},
				Status: FolderStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want: "0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.GetID()
			if got != tc.want {
				t.Errorf("GetID() got \n%s\n want \n%s", got, tc.want)
			}
		})
	}
}
