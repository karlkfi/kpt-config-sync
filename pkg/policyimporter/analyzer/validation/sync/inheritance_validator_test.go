package sync

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type inheritanceDisabledTestCase struct {
	name   string
	fgvkhs FileGroupVersionKindHierarchySync
	error  []string
}

func withMode(gvk schema.GroupVersionKind, mode v1alpha1.HierarchyModeType) FileGroupVersionKindHierarchySync {
	return FileGroupVersionKindHierarchySync{
		groupVersionKind: gvk,
		HierarchyMode:    mode,
	}
}

var inheritanceDisabledTestCases = []inheritanceDisabledTestCase{
	{
		name:   "no-inheritance rolebinding default",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeDefault),
	},
	{
		name:   "no-inheritance rolebinding quota error",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeHierarchicalQuota),
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
	},
	{
		name:   "no-inheritance rolebinding inherit error",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeInherit),
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
	},
	{
		name:   "no-inheritance rolebinding none error",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeNone),
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
	},
	{
		name:   "no-inheritance resourcequota default",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeDefault),
	},
	{
		name:   "no-inheritance resourcequota quota error",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeHierarchicalQuota),
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
	},
	{
		name:   "no-inheritance resourcequota inherit error",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeInherit),
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
	},
	{
		name:   "no-inheritance resourcequota none error",
		fgvkhs: withMode(kinds.ResourceQuota(), v1alpha1.HierarchyModeNone),
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
	},
}

func (tc inheritanceDisabledTestCase) Run(t *testing.T) {
	repo := v1alpha1.Repo{Spec: v1alpha1.RepoSpec{ExperimentalInheritance: false}}
	v := NewInheritanceValidatorFactory(&repo)

	syncs := []FileSync{toFileSync(tc.fgvkhs)}
	eb := multierror.Builder{}
	v.New(syncs).Validate(&eb)

	veterrorstest.ExpectErrors(tc.error, eb.Build(), t)
}

func TestInheritanceDisabledValidator(t *testing.T) {
	for _, tc := range inheritanceDisabledTestCases {
		t.Run(tc.name, tc.Run)
	}
}

type inheritanceEnabledTestCase struct {
	name   string
	fgvkhs FileGroupVersionKindHierarchySync
	error  []string
}

var inheritanceEnabledTestCases = []inheritanceEnabledTestCase{
	{
		name:   "inheritance rolebinding default",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeDefault),
	},
	{
		name:   "inheritance rolebinding quota error",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeHierarchicalQuota),
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
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
		error:  []string{veterrors.IllegalHierarchyModeErrorCode},
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

func (tc inheritanceEnabledTestCase) Run(t *testing.T) {
	repo := v1alpha1.Repo{Spec: v1alpha1.RepoSpec{ExperimentalInheritance: true}}
	v := NewInheritanceValidatorFactory(&repo)

	syncs := []FileSync{toFileSync(tc.fgvkhs)}
	eb := multierror.Builder{}
	v.New(syncs).Validate(&eb)

	veterrorstest.ExpectErrors(tc.error, eb.Build(), t)
}

func TestInheritanceEnabledValidator(t *testing.T) {
	for _, tc := range inheritanceEnabledTestCases {
		t.Run(tc.name, tc.Run)
	}
}

type inheritanceMissingTestCase struct {
	name   string
	fgvkhs FileGroupVersionKindHierarchySync
}

var inheritanceMissingTestCases = []inheritanceMissingTestCase{
	{
		name:   "inheritance rolebinding default",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeDefault),
	},
	{
		name:   "inheritance rolebinding quota error",
		fgvkhs: withMode(kinds.RoleBinding(), v1alpha1.HierarchyModeHierarchicalQuota),
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
}

func (tc inheritanceMissingTestCase) Run(t *testing.T) {
	v := NewInheritanceValidatorFactory(nil)

	syncs := []FileSync{toFileSync(tc.fgvkhs)}
	eb := multierror.Builder{}
	v.New(syncs).Validate(&eb)

	veterrorstest.ExpectErrors([]string{}, eb.Build(), t)
}

func TestNilRepo(t *testing.T) {
	for _, tc := range inheritanceMissingTestCases {
		t.Run(tc.name, tc.Run)
	}
}
