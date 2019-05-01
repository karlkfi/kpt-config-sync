package differ

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func namespaceConfig(opts ...object.Mutator) *v1.NamespaceConfig {
	return fake.Build(kinds.NamespaceConfig(), opts...).Object.(*v1.NamespaceConfig)
}

func markForDeletion(nsConfig *v1.NamespaceConfig) *v1.NamespaceConfig {
	nsConfig.Spec.DeleteSyncedTime = metav1.Now()
	return nsConfig
}

func namespace(opts ...object.Mutator) *corev1.Namespace {
	return fake.Build(kinds.Namespace(), opts...).Object.(*corev1.Namespace)
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
			actual:     namespace(),
			expectType: Update,
		},
		{
			name:       "in both, update even though cluster has invalid annotation",
			declared:   namespaceConfig(),
			actual:     namespace(managementInvalid),
			expectType: Update,
		},
		{
			name:     "in both, management disabled unmanage",
			declared: namespaceConfig(disableManaged),
			actual:   namespace(enableManaged),

			expectType: Unmanage,
		},
		{
			name:       "in both, management disabled noop",
			declared:   namespaceConfig(disableManaged),
			actual:     namespace(),
			expectType: NoOp,
		},
		{
			name:       "if not in repo but managed in cluster, delete",
			actual:     namespace(enableManaged),
			expectType: Delete,
		},
		{
			name:       "delete",
			declared:   markForDeletion(namespaceConfig()),
			actual:     namespace(enableManaged),
			expectType: Delete,
		},
		{
			name:       "in cluster only, unset noop",
			actual:     namespace(),
			expectType: NoOp,
		},
		{
			name:       "in cluster only, remove invalid management",
			actual:     namespace(managementInvalid),
			expectType: Unmanage,
		},
		{
			name:       "in cluster only, remove quota",
			actual:     namespace(object.Label(v1.ConfigManagementQuotaKey, "")),
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
