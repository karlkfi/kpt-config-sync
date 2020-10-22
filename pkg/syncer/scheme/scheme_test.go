package scheme

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/util/discovery"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestResourceScopes(t *testing.T) {
	testCases := []struct {
		name                string
		gvks                map[schema.GroupVersionKind]bool
		wantNamespacedTypes map[schema.GroupVersionKind]runtime.Object
		wantClusterTypes    map[schema.GroupVersionKind]runtime.Object
		wantErr             bool
	}{
		{
			name: "Ignore CustomResourceDefinition v1beta1",
			gvks: map[schema.GroupVersionKind]bool{
				kinds.CustomResourceDefinitionV1Beta1(): true,
			},
		},
		{
			name: "Ignore CustomResourceDefinition v1",
			gvks: map[schema.GroupVersionKind]bool{
				kinds.CustomResourceDefinitionV1Beta1().GroupKind().WithVersion("v1"): true,
			},
		},
		{
			name: "ClusterRole is cluster-scoped",
			gvks: map[schema.GroupVersionKind]bool{
				kinds.ClusterRole(): true,
			},
			wantClusterTypes: map[schema.GroupVersionKind]runtime.Object{
				kinds.ClusterRole(): &rbacv1.ClusterRole{},
			},
		},
		{
			name: "Role is namespaced",
			gvks: map[schema.GroupVersionKind]bool{
				kinds.Role(): true,
			},
			wantNamespacedTypes: map[schema.GroupVersionKind]runtime.Object{
				kinds.Role(): &rbacv1.Role{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := runtime.NewScheme()
			s.AddKnownTypeWithName(kinds.CustomResourceDefinitionV1Beta1(), &v1beta1.CustomResourceDefinition{})
			s.AddKnownTypeWithName(kinds.CustomResourceDefinitionV1Beta1().GroupKind().WithVersion("v1"), &unstructured.Unstructured{})
			err := rbacv1.AddToScheme(s)
			if err != nil {
				t.Fatal(err)
			}

			ns, cluster, err := ResourceScopes(tc.gvks, s, discovery.CoreScoper())

			if diff := cmp.Diff(tc.wantNamespacedTypes, ns, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
			if diff := cmp.Diff(tc.wantClusterTypes, cluster, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
			if err != nil && !tc.wantErr {
				t.Errorf("got error %s, want nil error", err)
			} else if err == nil && tc.wantErr {
				t.Error("got nil error, want error")
			}
		})
	}
}
