package differ

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cli-utils/pkg/common"
)

func namespaceConfig(opts ...core.MetaMutator) *v1.NamespaceConfig {
	result := fake.NamespaceConfigObject(opts...)
	return result
}

func markForDeletion(nsConfig *v1.NamespaceConfig) *v1.NamespaceConfig {
	nsConfig.Spec.DeleteSyncedTime = metav1.Now()
	return nsConfig
}

var (
	disableManaged    = syncertest.ManagementDisabled
	managementInvalid = core.Annotation(v1.ResourceManagementKey, "invalid")
	preventDeletion   = core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)
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
			actual:   fake.NamespaceObject("foo", syncertest.ManagementEnabled),

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
			actual:     fake.NamespaceObject("foo", syncertest.ManagementEnabled),
			expectType: NoOp,
		},
		{
			name:       "delete",
			declared:   markForDeletion(namespaceConfig()),
			actual:     fake.NamespaceObject("foo", syncertest.ManagementEnabled),
			expectType: Delete,
		},
		{
			name:       "marked for deletion, unmanage if deletion: prevent",
			declared:   markForDeletion(namespaceConfig()),
			actual:     fake.NamespaceObject("foo", syncertest.ManagementEnabled, preventDeletion),
			expectType: UnmanageNamespace,
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
