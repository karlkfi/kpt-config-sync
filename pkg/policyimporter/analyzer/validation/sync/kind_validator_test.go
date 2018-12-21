package sync

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vettesting "github.com/google/nomos/pkg/policyimporter/analyzer/vet/testing"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/rbac"
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
		gvk:  roleBinding(),
	},
	{
		name:  "crd not supported",
		gvk:   customResourceDefinition(),
		error: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		name:  "namespace not supported",
		gvk:   namespace(),
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

// resourceQuota is temporary. It is moved and documented in the already-approved next CL.
func resourceQuota() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("ResourceQuota")
}

// roleBinding is temporary. It is moved and documented in the already-approved next CL.
func roleBinding() schema.GroupVersionKind {
	return rbac.SchemeGroupVersion.WithKind("RoleBinding")
}

func TestKindValidator(t *testing.T) {
	for _, tc := range kindValidatorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
