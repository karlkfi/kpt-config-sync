package differ

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func namespaceConfig(opts ...object.MetaMutator) *v1.NamespaceConfig {
	result := fake.NamespaceConfigObject(fake.NamespaceConfigMeta(opts...))
	return result
}

func markForDeletion(nsConfig *v1.NamespaceConfig) *v1.NamespaceConfig {
	nsConfig.Spec.DeleteSyncedTime = metav1.Now()
	return nsConfig
}

var (
	enableManaged     = object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)
	disableManaged    = object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)
	managementInvalid = object.Annotation(v1.ResourceManagementKey, "invalid")
)

func TestNamespaceDiffType(t *testing.T) {
	testCases := []struct {
		name       string
		declared   *v1.NamespaceConfig
		actual     *corev1.Namespace
		expectType Type
	}{
		{
			name:       "in repo, create",
			declared:   namespaceConfig(),
			expectType: Create,
		},
		{
			name:       "in repo only and unmanaged, noop",
			declared:   namespaceConfig(disableManaged),
			expectType: NoOp,
		},
		{
			name:       "in repo only, management invalid error",
			declared:   namespaceConfig(managementInvalid),
			expectType: Error,
		},
		{
			name:       "in both, update",
			declared:   namespaceConfig(),
			actual:     fake.NamespaceObject("foo"),
			expectType: Update,
		},
		{
			name:       "in both, update even though cluster has invalid annotation",
			declared:   namespaceConfig(),
			actual:     fake.NamespaceObject("foo", managementInvalid),
			expectType: Update,
		},
		{
			name:     "in both, management disabled unmanage",
			declared: namespaceConfig(disableManaged),
			actual:   fake.NamespaceObject("foo", enableManaged),

			expectType: Unmanage,
		},
		{
			name:       "in both, management disabled noop",
			declared:   namespaceConfig(disableManaged),
			actual:     fake.NamespaceObject("foo"),
			expectType: NoOp,
		},
		{
			name:       "if not in repo but managed in cluster, noop",
			actual:     fake.NamespaceObject("foo", enableManaged),
			expectType: NoOp,
		},
		{
			name:       "delete",
			declared:   markForDeletion(namespaceConfig()),
			actual:     fake.NamespaceObject("foo", enableManaged),
			expectType: Delete,
		},
		{
			name:       "in cluster only, unset noop",
			actual:     fake.NamespaceObject("foo"),
			expectType: NoOp,
		},
		{
			name:       "in cluster only, remove invalid management",
			actual:     fake.NamespaceObject("foo", managementInvalid),
			expectType: Unmanage,
		},
		{
			name:       "in cluster only, remove quota",
			actual:     fake.NamespaceObject("foo", object.Label(v1.ConfigManagementQuotaKey, "")),
			expectType: Unmanage,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff := NamespaceDiff{
				Declared: tc.declared,
				Actual:   tc.actual,
			}

			if d := cmp.Diff(tc.expectType, diff.Type()); d != "" {
				t.Fatal(d)
			}
		})
	}
}
