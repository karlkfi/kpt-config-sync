package sync

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type kindValidatorTestCase struct {
	name  string
	gvk   schema.GroupVersionKind
	error []string
}

var kindValidatorTestCases = []kindValidatorTestCase{
	{
		name: "supported",
		gvk:  schema.GroupVersionKind{Group: "group"},
	},
	{
		name: "RoleBinding supported",
		gvk:  kinds.RoleBinding(),
	},
	{
		name:  "crd not supported",
		gvk:   kinds.CustomResourceDefinition(),
		error: []string{veterrors.UnsupportedResourceInSyncErrorCode},
	},
	{
		name:  "namespace not supported",
		gvk:   kinds.Namespace(),
		error: []string{veterrors.UnsupportedResourceInSyncErrorCode},
	},
	{
		name:  "nomos.dev group not supported",
		gvk:   schema.GroupVersionKind{Group: policyhierarchy.GroupName},
		error: []string{veterrors.UnsupportedResourceInSyncErrorCode},
	},
}

func (tc kindValidatorTestCase) Run(t *testing.T) {
	syncs := []FileSync{
		toFileSync(FileGroupVersionKindHierarchySync{groupVersionKind: tc.gvk}),
	}
	eb := multierror.Builder{}

	KindValidatorFactory.New(syncs).Validate(&eb)

	veterrorstest.ExpectErrors(tc.error, eb.Build(), t)
}

func TestKindValidator(t *testing.T) {
	for _, tc := range kindValidatorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
