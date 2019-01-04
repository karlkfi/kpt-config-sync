package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespaceTestCase struct {
	name      string
	namespace string
	error     []string
}

var namespaceTestCases = []namespaceTestCase{
	{
		name:      "no namespace",
		namespace: "",
	},
	{
		name:      "has namespace",
		namespace: "bar",
		error:     []string{veterrors.IllegalNamespaceDeclarationErrorCode},
	},
}

func (tc namespaceTestCase) Run(t *testing.T) {
	meta := resourceMeta{meta: &v1.ObjectMeta{Namespace: tc.namespace}}

	eb := multierror.Builder{}
	NamespaceValidatorFactory.New([]ResourceMeta{meta}).Validate(&eb)

	veterrorstest.ExpectErrors(tc.error, eb.Build(), t)
}

func TestNamespaceValidator(t *testing.T) {
	for _, tc := range namespaceTestCases {
		t.Run(tc.name, tc.Run)
	}
}
