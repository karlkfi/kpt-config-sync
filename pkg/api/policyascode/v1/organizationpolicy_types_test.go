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

func TestStorageOrganizationPolicy(t *testing.T) {
	key := types.NamespacedName{Name: "foo", Namespace: "default"}
	created := &OrganizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Spec: OrganizationPolicySpec{
			ResourceReference: ResourceReference{Kind: "Organization", Name: "bar"},
			Constraints: []OrganizationPolicyConstraint{
				{
					Constraint: "c1",
					ListPolicy: OrganizationPolicyListPolicy{
						AllowedValues:     []string{"projects/foo", "projects/bar"},
						DisallowedValues:  []string{"projects/disallowed-bar"},
						InheritFromParent: true,
						AllValues:         "ALLOW",
					},
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
			ImportDetails: fakeImportDetails,
		},
		Status: OrganizationPolicyStatus{
			SyncDetails: fakeSyncDetails,
		},
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
