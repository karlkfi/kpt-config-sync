package sync

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type versionTestCase struct {
	name  string
	gvks  []schema.GroupVersionKind
	error []string
}

var versionTestCases = []versionTestCase{
	{
		name: "empty",
	},
	{
		name: "one GVK",
		gvks: []schema.GroupVersionKind{
			{Group: "G1", Version: "V1", Kind: "K1"},
		},
	},
	{
		name: "two GVKs different Group",
		gvks: []schema.GroupVersionKind{
			{Group: "G1", Version: "V1", Kind: "K1"},
			{Group: "G2", Version: "V1", Kind: "K1"},
		},
	},
	{
		name: "two GVKs different Version",
		gvks: []schema.GroupVersionKind{
			{Group: "G1", Version: "V1", Kind: "K1"},
			{Group: "G1", Version: "V2", Kind: "K1"},
		},
		error: []string{veterrors.DuplicateSyncGroupKindErrorCode},
	},
	{
		name: "two GVKs different Kind",
		gvks: []schema.GroupVersionKind{
			{Group: "G1", Version: "V1", Kind: "K1"},
			{Group: "G1", Version: "V1", Kind: "K2"},
		},
	},
	{
		name: "three GVKs different Version, one error",
		gvks: []schema.GroupVersionKind{
			{Group: "G1", Version: "V1", Kind: "K1"},
			{Group: "G1", Version: "V2", Kind: "K1"},
			{Group: "G1", Version: "V3", Kind: "K1"},
		},
		error: []string{veterrors.DuplicateSyncGroupKindErrorCode},
	},
}

func (tc versionTestCase) Run(t *testing.T) {
	v := VersionValidatorFactory{}

	syncs := make([]FileSync, len(tc.gvks))
	for i, gvk := range tc.gvks {
		syncs[i] = toFileSync(FileGroupVersionKindHierarchySync{groupVersionKind: gvk})
	}

	eb := multierror.Builder{}
	v.New(syncs).Validate(&eb)

	veterrorstest.ExpectErrors(tc.error, eb.Build(), t)
}

func TestVersionValidation(t *testing.T) {
	for _, tc := range versionTestCases {
		t.Run(tc.name, tc.Run)
	}
}
