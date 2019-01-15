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

func TestStorageOrganization(t *testing.T) {
	key := types.NamespacedName{Name: "foo"}
	created := &Organization{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec: OrganizationSpec{
			ID: 1,
		},
		Status: OrganizationStatus{},
	}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &Organization{}
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

func TestOrganizationTFResourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		o       *Organization
		want    string
		wantErr bool
	}{
		{
			name: "Organization with valid ID",
			o: &Organization{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec: OrganizationSpec{
					ID: 1234567,
				},
				Status: OrganizationStatus{},
			},
			want: `data "google_organization" "bespin_organization" {
organization = "organizations/1234567"
}`,
			wantErr: false,
		},
		{
			name: "Organization with no ID",
			o: &Organization{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Spec:       OrganizationSpec{},
				Status:     OrganizationStatus{},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.o.TFResourceConfig(context.Background(), nil, nil)
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

func TestOrganizationID(t *testing.T) {
	tests := []struct {
		name string
		o    *Organization
		want string
	}{
		{
			name: "Organization with valid ID",
			o: &Organization{
				Spec: OrganizationSpec{
					ID: 1234567,
				},
			},
			want: "1234567",
		},
		{
			name: "Organization with empty ID",
			o: &Organization{
				Spec: OrganizationSpec{},
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.o.ID()
			if got != tc.want {
				t.Errorf("ID() got \n%s\n want \n%s", got, tc.want)
			}
		})
	}
}
