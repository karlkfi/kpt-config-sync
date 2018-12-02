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

func TestStorageIAMPolicy(t *testing.T) {
	key := types.NamespacedName{Name: "foo", Namespace: "default"}
	testCases := []*IAMPolicy{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
			Spec: IAMPolicySpec{
				ResourceReference: ResourceReference{Kind: OrganizationKind, Name: "bar"},
				Bindings:          fakeBindings,
				ImportDetails:     fakeImportDetails,
			},
			Status: IAMPolicyStatus{
				SyncDetails: fakeSyncDetails,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
			Spec: IAMPolicySpec{
				ResourceReference: ResourceReference{Kind: FolderKind, Name: "bar"},
				Bindings:          fakeBindings,
				ImportDetails:     fakeImportDetails,
			},
			Status: IAMPolicyStatus{
				SyncDetails: fakeSyncDetails,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
			Spec: IAMPolicySpec{
				ResourceReference: ResourceReference{Kind: ProjectKind, Name: "bar"},
				Bindings:          fakeBindings,
				ImportDetails:     fakeImportDetails,
			},
			Status: IAMPolicyStatus{
				SyncDetails: fakeSyncDetails,
			},
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

func TestIAMPolicyGetTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		ip      *IAMPolicy
		want    string
		wantErr bool
	}{
		{
			name: "IAMPolicy for Project",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
					ResourceReference: ResourceReference{
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
					ImportDetails: fakeImportDetails,
				},
				Status: IAMPolicyStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want: `resource "google_project_iam_policy" "project_iam_policy" {
project = "bar"
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
			name: "IAMPolicy with invalid ResourceReference",
			ip: &IAMPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: IAMPolicySpec{
					ResourceReference: ResourceReference{
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
					ImportDetails: fakeImportDetails,
				},
				Status: IAMPolicyStatus{
					SyncDetails: fakeSyncDetails,
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.ip.GetTFResourceConfig()
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
