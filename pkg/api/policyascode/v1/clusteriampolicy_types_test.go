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

func TestStorageClusterIAMPolicy(t *testing.T) {
	key := types.NamespacedName{Name: "foo"}
	testCases := []*ClusterIAMPolicy{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			Spec: IAMPolicySpec{
				ResourceRef: corev1.ObjectReference{Kind: OrganizationKind, Name: "bar"},
				Bindings:    fakeBindings,
			},
			Status: IAMPolicyStatus{},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			Spec: IAMPolicySpec{
				ResourceRef: corev1.ObjectReference{Kind: FolderKind, Name: "bar"},
				Bindings:    fakeBindings,
			},
			Status: IAMPolicyStatus{},
		},
	}

	g := gomega.NewGomegaWithT(t)
	for _, created := range testCases {
		// Test Create
		fetched := &ClusterIAMPolicy{}
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
func TestClusterIAMPolicyTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		ip      *ClusterIAMPolicy
		c       *stubClient
		want    string
		wantErr bool
	}{
		{
			name: "ClusterIAMPolicy for Organization",
			ip: &ClusterIAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: OrganizationKind,
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
"user:member2@bar.com"
]}
binding {
role = "roles/iam.organizationRoleAdmin"
members = [
"serviceAccount:service-account@foo.com"
]}
}`,
		},
		{
			name: "ClusterIAMPolicy for Organization with empty bindings",
			ip: &ClusterIAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: OrganizationKind,
						Name: "bar",
					},
					Bindings: []IAMPolicyBinding{},
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
policy_data = "{}"
}
`,
		},
		{
			name: "ClusterIAMPolicy for Folder",
			ip: &ClusterIAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: FolderKind,
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
"user:member2@bar.com"
]}
binding {
role = "roles/resourcemanager.folderAdmin"
members = [
"serviceAccount:service-account@foo.com"
]}
}`,
		},
		{
			name: "ClusterIAMPolicy with invalid Project reference",
			ip: &ClusterIAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: ProjectKind,
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
			wantErr: true,
		},
		{
			name: "ClusterIAMPolicy for Organization, but missing Organization ID",
			ip: &ClusterIAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: OrganizationKind,
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
					// No ID.
					Spec: OrganizationSpec{},
				},
			},
			wantErr: true,
		},
		{
			name: "ClusterIAMPolicy for Folder, but missing Folder ID",
			ip: &ClusterIAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: IAMPolicySpec{
					ResourceRef: corev1.ObjectReference{
						Kind: FolderKind,
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
					// No ID.
					Spec: FolderSpec{},
				},
			},
			wantErr: true,
		},
		{
			name: "ClusterIAMPolicy with invalid ResourceReference",
			ip: &ClusterIAMPolicy{
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
			wantErr: true,
		},
		{
			name: "ClusterIAMPolicy with No ResourceReference",
			ip: &ClusterIAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
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
			c:       &stubClient{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.ip.TFResourceConfig(context.Background(), tc.c, nil)
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
