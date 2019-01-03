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
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestQuoteList(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{
			name:   "single element",
			values: []string{"foo"},
			want:   `["foo"]`,
		},
		{
			name:   "empty list",
			values: []string{},
			want:   "[]",
		},
		{
			name:   "multiple elements",
			values: []string{"foo", "bar", "baz"},
			want:   `["foo", "bar", "baz"]`,
		},
		{
			name:   "elements with quoted values",
			values: []string{"foo", `bar"`, `baz\`},
			want:   `["foo", "bar\"", "baz\\"]`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := quoteList(tc.values)
			if got != tc.want {
				t.Errorf("quoteList got %v, wanted %v", got, tc.want)
			}
		})
	}
}
func TestOrganizationPolicyGetTFResourceConfig(t *testing.T) {
	org := &Organization{
		ObjectMeta: metav1.ObjectMeta{Name: "foo"},
		Spec:       OrganizationSpec{},
		Status:     OrganizationStatus{},
	}
	folder := &Folder{
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
	project := &Project{
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

	tests := []struct {
		name    string
		obj     runtime.Object
		ops     OrganizationPolicy
		want    string
		wantErr bool
	}{
		{
			name: "policy with allowed values",
			obj:  org,
			ops: OrganizationPolicy{
				Spec: OrganizationPolicySpec{
					ResourceRef: corev1.ObjectReference{Kind: OrganizationKind, Name: "bar"},
					Constraints: []OrganizationPolicyConstraint{
						{
							Constraint: "c1",
							ListPolicy: OrganizationPolicyListPolicy{
								AllowedValues: []string{"projects/foo", "projects/bar"},
							},
						},
					},
				},
			},
			want: `["projects/foo", "projects/bar"]`,
		},
		{
			name: "policy with disallowed values",
			obj:  folder,
			ops: OrganizationPolicy{
				Spec: OrganizationPolicySpec{
					ResourceRef: corev1.ObjectReference{Kind: FolderKind, Name: "bar"},
					Constraints: []OrganizationPolicyConstraint{
						{
							Constraint: "c1",
							ListPolicy: OrganizationPolicyListPolicy{
								DisallowedValues: []string{"projects/disallowed-bar"},
							},
						},
					},
				},
			},
			want: `values = ["projects/disallowed-bar"]`,
		},
		{
			name: "policy with boolean value",
			obj:  project,
			ops: OrganizationPolicy{
				Spec: OrganizationPolicySpec{
					ResourceRef: corev1.ObjectReference{Kind: ProjectKind, Name: "bar"},
					Constraints: []OrganizationPolicyConstraint{
						{
							Constraint:    "c3",
							BooleanPolicy: OrganizationPolicyBooleanPolicy{Enforced: true},
						},
					},
				},
			},
			want: "enforced = true",
		},
		{
			name: "policy with allow all value",
			obj:  org,
			ops: OrganizationPolicy{
				Spec: OrganizationPolicySpec{
					ResourceRef: corev1.ObjectReference{Kind: OrganizationKind, Name: "bar"},
					Constraints: []OrganizationPolicyConstraint{
						{
							Constraint: "c3",
							ListPolicy: OrganizationPolicyListPolicy{
								AllValues: "ALLOW",
							},
						},
					},
				},
			},
			want: "allow",
		},
		{
			name: "policy with deny all value",
			obj:  org,
			ops: OrganizationPolicy{
				Spec: OrganizationPolicySpec{
					ResourceRef: corev1.ObjectReference{Kind: OrganizationKind, Name: "bar"},
					Constraints: []OrganizationPolicyConstraint{
						{
							Constraint: "c3",
							ListPolicy: OrganizationPolicyListPolicy{
								AllValues: "DENY",
							},
						},
					},
				},
			},
			want: "deny",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := stubClient{obj: tc.obj}
			got, err := tc.ops.TFResourceConfig(context.Background(), &client)
			switch {
			case !tc.wantErr && err != nil:
				t.Errorf("TFResourceConfig() got err %+v; want nil", err)
			case tc.wantErr && err == nil:
				t.Errorf("TFResourceConfig() got nil; want err %+v", tc.wantErr)
			case !strings.Contains(got, tc.want):
				t.Errorf("TFResourceConfig() got [%s] does not contain [%s]", got, tc.want)
			}

		})
	}
}

func TestStorageOrganizationPolicy(t *testing.T) {
	key := types.NamespacedName{Name: "foo", Namespace: "default"}
	created := &OrganizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Spec: OrganizationPolicySpec{
			ResourceRef: corev1.ObjectReference{Kind: OrganizationKind, Name: "bar"},
			Constraints: []OrganizationPolicyConstraint{
				{
					Constraint: "c1",
					ListPolicy: OrganizationPolicyListPolicy{
						AllowedValues:     []string{"projects/foo", "projects/bar"},
						DisallowedValues:  []string{"projects/disallowed-bar"},
						InheritFromParent: true,
						AllValues:         "ALLOW",
					},
					// TODO(lschumacher): this spec is ill-formed
					BooleanPolicy: OrganizationPolicyBooleanPolicy{Enforced: true},
				},
				{
					Constraint: "c2",
					ListPolicy: OrganizationPolicyListPolicy{
						AllowedValues:     []string{"projects/bar"},
						DisallowedValues:  []string{"projects/disallowed-foo", "projects/disallowed-bar"},
						InheritFromParent: false,
						AllValues:         "DENY",
					},
					BooleanPolicy: OrganizationPolicyBooleanPolicy{Enforced: false},
				},
			},
		},
		Status: OrganizationPolicyStatus{},
	}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &OrganizationPolicy{}
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
