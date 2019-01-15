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

func TestStorageProject(t *testing.T) {
	key := types.NamespacedName{Name: "foo", Namespace: "default"}
	created := &Project{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Spec: ProjectSpec{
			ParentRef: corev1.ObjectReference{
				Kind: FolderKind,
				Name: "bar",
			},
			DisplayName: "spec-bar",
			ID:          "some-fake-project",
		},
		Status: ProjectStatus{},
	}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &Project{}
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

func TestProjectTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		p       *Project
		c       *stubClient
		want    string
		wantErr bool
	}{
		{
			name: "Project with Organization as parent",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: ProjectSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Organization",
						Name: "bar",
					},
					DisplayName: "spec-bar",
					ID:          "some-fake-project",
				},
				Status: ProjectStatus{},
			},
			c: &stubClient{
				obj: &Organization{
					Spec: OrganizationSpec{
						ID: 1234567,
					},
				},
			},
			want: `resource "google_project" "bespin_project" {
name = "spec-bar"
project_id = "some-fake-project"
org_id = "1234567"
}`,
			wantErr: false,
		},
		{
			name: "Project with Folder as parent",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{Name: "foo",
					Namespace: "default",
				},
				Spec: ProjectSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Folder",
						Name: "bar",
					},
					DisplayName: "spec-bar",
					ID:          "some-fake-project",
				},
				Status: ProjectStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			want: `resource "google_project" "bespin_project" {
name = "spec-bar"
project_id = "some-fake-project"
folder_id = "1234567"
}`,
			wantErr: false,
		},
		{
			name: "Project with Organization as parent, but missing parent Organization ID",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: ProjectSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Organization",
						Name: "bar",
					},
					DisplayName: "spec-bar",
					ID:          "some-fake-project",
				},
				Status: ProjectStatus{},
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
			name: "Project with Folder as parent, but missing parent Folder ID",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{Name: "foo",
					Namespace: "default",
				},
				Spec: ProjectSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "Folder",
						Name: "bar",
					},
					DisplayName: "spec-bar",
					ID:          "some-fake-project",
				},
				Status: ProjectStatus{},
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
			name: "Project with parent name, but missing parent kind",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: ProjectSpec{
					ParentRef: corev1.ObjectReference{
						Kind: "",
						Name: "bar",
					},
					DisplayName: "spec-bar",
					ID:          "some-fake-project",
				},
				Status: ProjectStatus{},
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
			name: "Project with no parent reference",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: ProjectSpec{
					DisplayName: "spec-bar",
					ID:          "some-fake-project",
				},
				Status: ProjectStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			want: `resource "google_project" "bespin_project" {
name = "spec-bar"
project_id = "some-fake-project"

}`,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.p.TFResourceConfig(context.Background(), tc.c, nil)
			switch {
			case !tc.wantErr && err != nil:
				t.Errorf("TFResourceConfig() got err %+v; want nil", err)
			case tc.wantErr && err == nil:
				t.Errorf("TFResourceConfig() got nil; want err %+v", tc.wantErr)
			case got != tc.want:
				t.Errorf("TFResourceConfig() got %s; want %s", got, tc.want)
			}
		})
	}
}
