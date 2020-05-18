package discovery

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAPIInfo_GroupVersionKinds(t *testing.T) {
	testCases := []struct {
		name          string
		resourceLists []*metav1.APIResourceList
		syncs         []*v1.Sync
		expected      map[schema.GroupVersionKind]bool
	}{
		{
			name: "Lists only mentioned gks",
			resourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "rbac/v1",
					APIResources: []metav1.APIResource{
						{
							Kind: "Role",
						},
					},
				},
				{
					GroupVersion: "rbac/v2",
					APIResources: []metav1.APIResource{
						{
							Kind: "Role",
						},
					},
				},
				{
					GroupVersion: "apps/v1",
					APIResources: []metav1.APIResource{
						{
							Kind: "Deployment",
						},
					},
				},
			},
			syncs: []*v1.Sync{
				{
					Spec: v1.SyncSpec{
						Group: "rbac",
						Kind:  "Role",
					},
				},
			},
			expected: map[schema.GroupVersionKind]bool{
				{Group: "rbac", Version: "v1", Kind: "Role"}: true,
				{Group: "rbac", Version: "v2", Kind: "Role"}: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			api, err := NewAPIInfo(tc.resourceLists)
			if err != nil {
				t.Error(err)
			}

			result := api.GroupVersionKinds(tc.syncs...)
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
