package sync

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type knownResourceValidatorTestCase struct {
	name  string
	known []schema.GroupVersionKind
	gvk   schema.GroupVersionKind
	error []string
}

var knownResourceValidatorTestCases = []knownResourceValidatorTestCase{
	{
		name:  "RoleBinding throws error if not known",
		gvk:   kinds.RoleBinding(),
		error: []string{vet.UnknownResourceInSyncErrorCode},
	},
	{
		name:  "RoleBinding valid if known",
		known: []schema.GroupVersionKind{kinds.RoleBinding()},
		gvk:   kinds.RoleBinding(),
	},
	{
		name:  "RoleBinding throws error if wrong version",
		gvk:   kinds.RoleBinding(),
		known: []schema.GroupVersionKind{kinds.RoleBinding().GroupKind().WithVersion("v2")},
		error: []string{vet.UnknownResourceVersionInSyncErrorCode},
	},
}

func toAPIInfo(known []schema.GroupVersionKind) (*discovery.APIInfo, error) {
	resources := make([]*metav1.APIResourceList, len(known))

	for i, gvk := range known {
		resources[i] = &metav1.APIResourceList{
			GroupVersion: gvk.GroupVersion().String(),
			APIResources: []metav1.APIResource{{Kind: gvk.Kind}},
		}
	}

	return discovery.NewAPIInfo(resources)
}

func (tc knownResourceValidatorTestCase) Run(t *testing.T) {
	syncs := []FileSync{
		toFileSync(FileGroupVersionKindHierarchySync{groupVersionKind: tc.gvk}),
	}
	eb := multierror.Builder{}

	apiInfo, err := toAPIInfo(tc.known)
	if err != nil {
		t.Fatalf("unexpected error forming APIInfo: %v", err)
	}

	KnownResourceValidatorFactory(apiInfo).New(syncs).Validate(&eb)

	vettesting.ExpectErrors(tc.error, eb.Build(), t)
}

func TestKnownResourceValidator(t *testing.T) {
	for _, tc := range knownResourceValidatorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
