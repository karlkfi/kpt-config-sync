package sync

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func withMode(gvk schema.GroupVersionKind, mode v1alpha1.HierarchyModeType) FileGroupVersionKindHierarchySync {
	return FileGroupVersionKindHierarchySync{
		groupVersionKind: gvk,
		HierarchyMode:    mode,
	}
}

type testCase struct {
	name   string
	fgvkhs FileGroupVersionKindHierarchySync
	error  []string
}

var testCases = []testCase{
	{
		name:   "inheritance rolebinding default",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeDefault),
	},
	{
		name:   "inheritance rolebinding quota error",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeHierarchicalQuota),
		error:  []string{vet.IllegalSyncHierarchyModeErrorCode},
	},
	{
		name:   "inheritance rolebinding inherit",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeInherit),
	},
	{
		name:   "inheritance rolebinding none",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeNone),
	},
	{
		name:   "inheritance resourcequota default",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeDefault),
	},
	{
		name:   "inheritance resourcequota quota",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeHierarchicalQuota),
	},
	{
		name:   "inheritance resourcequota inherit",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeInherit),
	},
	{
		name:   "inheritance resourcequota none",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeNone),
	},
	{
		name:   "inheritance resourcequota v2 default",
		fgvkhs: withMode(kinds.ResourceQuota().GroupKind().WithVersion("v2"), v1alpha1.HierarchyModeDefault),
	},
	{
		name:   "inheritance resourcequota v2 quota error",
		fgvkhs: withMode(kinds.ResourceQuota().GroupKind().WithVersion("v2"), v1alpha1.HierarchyModeHierarchicalQuota),
		error:  []string{vet.IllegalSyncHierarchyModeErrorCode},
	},
	{
		name:   "inheritance resourcequota v2 inherit",
		fgvkhs: withMode(kinds.ResourceQuota().GroupKind().WithVersion("v2"), v1alpha1.HierarchyModeInherit),
	},
	{
		name:   "inheritance resourcequota v2 none",
		fgvkhs: withMode(kinds.ResourceQuota().GroupKind().WithVersion("v2"), v1alpha1.HierarchyModeNone),
	},
}

func (tc testCase) Run(t *testing.T) {
	v := NewInheritanceValidatorFactory()

	syncs := []FileSync{toFileSync(tc.fgvkhs)}
	eb := multierror.Builder{}
	v.New(syncs).Validate(&eb)

	vettesting.ExpectErrors(tc.error, eb.Build(), t)
}

func TestValidator(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}
