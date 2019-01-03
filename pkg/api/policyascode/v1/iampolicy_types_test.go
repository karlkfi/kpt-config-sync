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

func TestStorageIAMPolicy(t *testing.T) {
	key := types.NamespacedName{Name: "foo", Namespace: "default"}
	testCases := []*IAMPolicy{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
			Spec: IAMPolicySpec{
				ResourceRef: corev1.ObjectReference{Kind: OrganizationKind, Name: "bar"},
				Bindings:    fakeBindings,
			},
			Status: IAMPolicyStatus{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
			Spec: IAMPolicySpec{
				ResourceRef: corev1.ObjectReference{Kind: FolderKind, Name: "bar"},
				Bindings:    fakeBindings,
			},
			Status: IAMPolicyStatus{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
			Spec: IAMPolicySpec{
				ResourceRef: corev1.ObjectReference{Kind: ProjectKind, Name: "bar"},
				Bindings:    fakeBindings,
			},
			Status: IAMPolicyStatus{},
		},
	}
	g := gomega.NewGomegaWithT(t)

	for _, created := range testCases {
		// Test Create
		fetched := &IAMPolicy{}
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
}

func TestIAMPolicyTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		ip      *IAMPolicy
		c       *stubClient
		want    string
		wantErr bool
	}{
		{
			name: "IAMPolicy for Organization",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Organization",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/iam.organizationRoleAdmin",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/iam.organizationRoleAdmin",
						},
					},
				},
				Status: IAMPolicyStatus{},
			},
			c: &stubClient{
				obj: &Organization{
					Spec: OrganizationSpec{
						ID: 1234567,
					},
				},
			},
			want: `resource "google_organization_iam_policy" "bespin_organization_iam_policy" {
org_id = "organizations/1234567"
policy_data = "${data.google_iam_policy.admin.policy_data}"
}
data "google_iam_policy" "admin" {
binding {
role = "roles/iam.organizationRoleAdmin"
members = [
"user:member1@foo.com",
"user:member2@bar.com",
]}
binding {
role = "roles/iam.organizationRoleAdmin"
members = [
"serviceAccount:service-account@foo.com",
]}
}`,
			wantErr: false,
		},
		{
			name: "IAMPolicy for Folder",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Folder",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/resourcemanager.folderAdmin",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/resourcemanager.folderAdmin",
						},
					},
				},
				Status: IAMPolicyStatus{},
			},
			c: &stubClient{
				obj: &Folder{
					Spec: FolderSpec{
						ID: 1234567,
					},
				},
			},
			want: `resource "google_folder_iam_policy" "bespin_folder_iam_policy" {
folder = "folders/1234567"
policy_data = "${data.google_iam_policy.admin.policy_data}"
}
data "google_iam_policy" "admin" {
binding {
role = "roles/resourcemanager.folderAdmin"
members = [
"user:member1@foo.com",
"user:member2@bar.com",
]}
binding {
role = "roles/resourcemanager.folderAdmin"
members = [
"serviceAccount:service-account@foo.com",
]}
}`,
			wantErr: false,
		},
		{
			name: "IAMPolicy for Project",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Project",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/editor",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/owner",
						},
					},
				},
				Status: IAMPolicyStatus{},
			},
			c: &stubClient{
				obj: &Project{
					Spec: ProjectSpec{
						ID: "project-001",
					},
				},
			},
			want: `resource "google_project_iam_policy" "bespin_project_iam_policy" {
project = "project-001"
policy_data = "${data.google_iam_policy.admin.policy_data}"
}
data "google_iam_policy" "admin" {
binding {
role = "roles/editor"
members = [
"user:member1@foo.com",
"user:member2@bar.com",
]}
binding {
role = "roles/owner"
members = [
"serviceAccount:service-account@foo.com",
]}
}`,
			wantErr: false,
		},
		{
			name: "IAMPolicy for Organization, but missing Organization ID",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Organization",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/iam.organizationRoleAdmin",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/iam.organizationRoleAdmin",
						},
					},
				},
				Status: IAMPolicyStatus{},
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
			name: "IAMPolicy for Folder, but missing Folder ID",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Folder",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/resourcemanager.folderAdmin",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/resourcemanager.folderAdmin",
						},
					},
				},
				Status: IAMPolicyStatus{},
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
			name: "IAMPolicy for Project, but missing Project ID",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Project",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/editor",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/owner",
						},
					},
				},
				Status: IAMPolicyStatus{},
			},
			c: &stubClient{
				obj: &Project{
					Spec: ProjectSpec{},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "IAMPolicy with invalid ResourceReference",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Invalid",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/editor",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/owner",
						},
					},
				},
				Status: IAMPolicyStatus{},
			},
			c: &stubClient{
				obj: &Project{
					Spec: ProjectSpec{
						ID: "project-001",
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "IAMPolicy with No ResourceReference",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: "Project",
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{
						{
							Members: []string{
								"user:member1@foo.com",
								"user:member2@bar.com",
							},
							Role: "roles/editor",
						},
						{
							Members: []string{
								"serviceAccount:service-account@foo.com",
							},
							Role: "roles/owner",
						},
					},
				},
				Status: IAMPolicyStatus{},
			},
			c: &stubClient{
				obj: &Project{
					Spec: ProjectSpec{},
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.ip.TFResourceConfig(context.Background(), tc.c)
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
