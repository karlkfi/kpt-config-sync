package sync

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vettesting "github.com/google/nomos/pkg/policyimporter/analyzer/vet/testing"
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
		error: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		name:  "namespace not supported",
		gvk:   kinds.Namespace(),
		error: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		name:  "nomos.dev group not supported",
		gvk:   schema.GroupVersionKind{Group: policyhierarchy.GroupName},
		error: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
}

func (tc kindValidatorTestCase) Run(t *testing.T) {
	syncs := []FileSync{
		toFileSync(FileGroupVersionKindHierarchySync{GroupVersionKind: tc.gvk}),
	}
	eb := multierror.Builder{}

	KindValidatorFactory.New(syncs).Validate(&eb)

	vettesting.ExpectErrors(tc.error, eb.Build(), t)
}

func TestKindValidator(t *testing.T) {
	for _, tc := range kindValidatorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
