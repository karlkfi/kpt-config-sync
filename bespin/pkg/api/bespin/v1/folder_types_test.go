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
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageFolder(t *testing.T) {
	key := types.NamespacedName{Name: "foo"}
	created := &Folder{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec: FolderSpec{
			ParentRef: corev1.ObjectReference{
				Kind: OrganizationKind,
				Name: "bar",
			},
			DisplayName: "spec-bar",
			ID:          1,
		},
		Status: FolderStatus{},
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

func TestFolderTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		f       *Folder
		c       *stubClient
		want    string
		tfState map[string]string
		wantErr bool
	}{
		{
			name: "Folder with Organization as parent",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Organization",
						Name: "organizations-001",
					},
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Organization{
					Spec: OrganizationSpec{
						ID: 1234567,
					},
				},
			},
			want: `resource "google_folder" "bespin_folder" {
display_name = "spec-bar"
parent = "organizations/1234567"
}`,
			wantErr: false,
		},
		{
			name: "Folder with Folder as parent",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Folder",
						Name: "folders-001",
					},
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			want: `resource "google_folder" "bespin_folder" {
display_name = "spec-bar"
parent = "folders/1234567"
}`,
			wantErr: false,
		},
		{
			name: "Folder with no parent reference, parent Organization is stored in terraform state",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					DisplayName: "spec-bar",
					ID:          1234567,
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			tfState: map[string]string{
				"parent": "organizations/7654321",
			},
			want: `resource "google_folder" "bespin_folder" {
display_name = "spec-bar"
parent = "organizations/7654321"
}`,
			wantErr: false,
		},
		{
			name: "Folder with no parent reference, parent Folder is stored in terraform state",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					DisplayName: "spec-bar",
					ID:          1234567,
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			tfState: map[string]string{
				"parent": "folders/7654321",
			},
			want: `resource "google_folder" "bespin_folder" {
display_name = "spec-bar"
parent = "folders/7654321"
}`,
			wantErr: false,
		},
		{
			name: "Folder with no parent reference, and there is no parent in terraform state",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					DisplayName: "spec-bar",
					ID:          1234567,
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Folder with Organization as parent, but missing parent Organizaton ID",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Organization",
						Name: "",
					},
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Organization{
					Spec: OrganizationSpec{},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Folder with Folder as parent, but missing parent Folder ID",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Folder",
						Name: "",
					},
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Folder missing parent reference kind, but with parent reference name",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "",
						Name: "bar",
					},
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Folder with invalid parent reference kind",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "invalid",
						Name: "bar",
					},
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.f.TFResourceConfig(context.Background(), tc.c, tc.tfState)
			switch {
			case !tc.wantErr && err != nil:
				t.Errorf("TFResourceConfig() got err %+v; want nil", err)
			case tc.wantErr && err == nil:
				t.Errorf("TFResourceConfig() got nil; want err %+v", tc.wantErr)
			case got != tc.want:
				t.Errorf("TFResourceConfig() got \n%s\n want \n%s", got, tc.want)
			}
		})
	}
}

func TestFolderID(t *testing.T) {
	tests := []struct {
		name string
		f    *Folder
		want string
	}{
		{
			name: "Folder with Organization as parent with valid Spec.ID",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Organization",
						Name: "organizations/1234567",
					},
					ID:          7654321,
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			want: "7654321",
		},
		{
			name: "Folder with Folder as parent, with valid Spec.ID",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Folder",
						Name: "folders/1234567",
					},
					DisplayName: "spec-bar",
					ID:          9876543,
				},
				Status: FolderStatus{},
			},
			want: "9876543",
		},
		{
			name: "Folder with no Spec.ID",
			f: &Folder{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: FolderSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Invalid",
						Name: "bar",
					},
					DisplayName: "spec-bar",
				},
				Status: FolderStatus{},
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.f.ID()
			if got != tc.want {
				t.Errorf("ID() got \n%s\n want \n%s", got, tc.want)
			}
		})
	}
}
