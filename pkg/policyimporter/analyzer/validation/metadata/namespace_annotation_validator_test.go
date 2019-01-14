package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type namespaceAnnotationTestCase struct {
	name        string
	gvk         schema.GroupVersionKind
	annotations map[string]string
	error       []string
}

func (tc namespaceAnnotationTestCase) Run(t *testing.T) {
	meta := resourceMeta{groupVersionKind: tc.gvk, meta: &v1.ObjectMeta{Annotations: tc.annotations}}

	eb := multierror.Builder{}
	NamespaceAnnotationValidatorFactory.New([]ResourceMeta{meta}).Validate(&eb)
	veterrorstest.ExpectErrors(tc.error, eb.Build(), t)
}

var namespaceAnnotationTestCases = []namespaceAnnotationTestCase{
	{
		name: "empty annotations",
		gvk:  kinds.Namespace(),
	},
	{
		name:        "legal annotation on Namespace",
		annotations: map[string]string{"annotation": "stuff"},
		gvk:         kinds.Namespace(),
	},
	{
		name:        "namespaceselector annotation on Namespace",
		annotations: map[string]string{v1alpha1.NamespaceSelectorAnnotationKey: "not legal"},
		gvk:         kinds.Namespace(),
		error:       []string{veterrors.IllegalNamespaceAnnotationErrorCode},
	},
	{
		name:        "namespaceselector annotation on Role",
		annotations: map[string]string{v1alpha1.NamespaceSelectorAnnotationKey: "fine since not namespace"},
		gvk:         kinds.Role(),
	},
	{
		name:        "legal and namespaceselector annotations on Namespace",
		annotations: map[string]string{"annotation": "stuff", v1alpha1.NamespaceSelectorAnnotationKey: "not legal"},
		gvk:         kinds.Namespace(),
		error:       []string{veterrors.IllegalNamespaceAnnotationErrorCode},
	},
}

func TestNamespaceAnnotationValidator_Validate(t *testing.T) {
	for _, tc := range namespaceAnnotationTestCases {
		t.Run(tc.name, tc.Run)
	}
}
