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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageProject(t *testing.T) {
	key := types.NamespacedName{Name: "foo", Namespace: "default"}
	created := &Project{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Spec: ProjectSpec{
			ParentReference: ParentReference{
				Kind: FolderKind,
				Name: "bar",
			},
			Name:          "spec-bar",
			ID:            "some-fake-project",
			ImportDetails: fakeImportDetails,
		},
		Status: ProjectStatus{
			SyncDetails: fakeSyncDetails,
		},
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

func TestProjectGetTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		p       *Project
		want    string
		wantErr bool
	}{
		{
			name: "Project with Organization as parent",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
					Annotations: map[string]string{
						"key1":                  "value1",
						ParentOrganizationIDKey: "1234567",
					},
				},
				Spec: ProjectSpec{
					ParentReference: ParentReference{
						Kind: "Organization",
						Name: "bar",
					},
					Name:          "spec-bar",
					ID:            "some-fake-project",
					ImportDetails: fakeImportDetails,
				},
				Status: ProjectStatus{
					SyncDetails: fakeSyncDetails,
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
					Annotations: map[string]string{
						"key1":            "value1",
						ParentFolderIDKey: "1234567",
					},
				},
				Spec: ProjectSpec{
					ParentReference: ParentReference{
						Kind: "Folder",
						Name: "bar",
					},
					Name:          "spec-bar",
					ID:            "some-fake-project",
					ImportDetails: fakeImportDetails,
				},
				Status: ProjectStatus{
					SyncDetails: fakeSyncDetails,
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
			name: "Project with invalid parentReference",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: ProjectSpec{
					ParentReference: ParentReference{
						Kind: "invalid",
						Name: "bar",
					},
					Name:          "spec-bar",
					ID:            "some-fake-project",
					ImportDetails: fakeImportDetails,
				},
				Status: ProjectStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Project with no Parent ID annotation",
			p: &Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
					Annotations: map[string]string{
						"key1": "value1",
					},
				},
				Spec: ProjectSpec{
					ParentReference: ParentReference{
						Kind: "Organization",
						Name: "bar",
					},
					Name:          "spec-bar",
					ID:            "some-fake-project",
					ImportDetails: fakeImportDetails,
				},
				Status: ProjectStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.p.GetTFResourceConfig()
			switch {
			case !tc.wantErr && err != nil:
				t.Errorf("GetTFResourceConfig() got err %+v; want nil", err)
			case tc.wantErr && err == nil:
				t.Errorf("GetTFResourceConfig() got nil; want err %+v", tc.wantErr)
			case got != tc.want:
				t.Errorf("GetTFResourceConfig() got %s; want %s", got, tc.want)
			}
		})
	}
}
