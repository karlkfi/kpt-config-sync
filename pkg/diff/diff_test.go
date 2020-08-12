package diff

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDiffType(t *testing.T) {
	testCases := []struct {
		name       string
		declared   core.Object
		actual     core.Object
		expectType Type
	}{
		{
			name:       "in repo, create",
			declared:   fake.ClusterRoleObject(),
			expectType: Create,
		},
		{
			name:       "in repo only and unmanaged, noop",
			declared:   fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
			expectType: NoOp,
		},
		{
			name:       "in repo only, management invalid error",
			declared:   fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "invalid")),
			expectType: Error,
		},
		{
			name:       "in repo only, management empty string error",
			declared:   fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "")),
			expectType: Error,
		},
		{
			name:       "in both, update",
			declared:   fake.ClusterRoleObject(),
			actual:     fake.ClusterRoleObject(),
			expectType: Update,
		},
		{
			name:       "in both and owned, update",
			declared:   fake.ClusterRoleObject(),
			actual:     fake.ClusterRoleObject(core.OwnerReference([]metav1.OwnerReference{{}})),
			expectType: Update,
		},
		{
			name:       "in both, update even though cluster has invalid annotation",
			declared:   fake.ClusterRoleObject(),
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "invalid")),
			expectType: Update,
		},
		{
			name:       "in both, management disabled unmanage",
			declared:   fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			expectType: Unmanage,
		},
		{
			name:       "in both, management disabled noop",
			declared:   fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
			actual:     fake.ClusterRoleObject(),
			expectType: NoOp,
		},
		{
			name:       "delete",
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			expectType: Delete,
		},
		{
			name:       "in cluster only, unset noop",
			actual:     fake.ClusterRoleObject(),
			expectType: NoOp,
		},
		{
			name:       "in cluster only, remove invalid empty string",
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "")),
			expectType: Unmanage,
		},
		{
			name:       "in cluster only, remove invalid annotation",
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "invalid")),
			expectType: Unmanage,
		},
		{
			name: "in cluster only and owned, do nothing",
			actual: fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.OwnerReference([]metav1.OwnerReference{{}})),
			expectType: NoOp,
		},
		{
			name: "in cluster only and owned and prevent deletion, unmanage",
			actual: fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.OwnerReference([]metav1.OwnerReference{{}}),
				core.Annotation(lifecycle.Deletion, lifecycle.PreventDeletion)),
			expectType: NoOp,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff := Diff{
				Declared: tc.declared,
				Actual:   tc.actual,
			}

			if d := cmp.Diff(tc.expectType, diff.Type()); d != "" {
				t.Fatal(d)
			}
		})
	}
}
