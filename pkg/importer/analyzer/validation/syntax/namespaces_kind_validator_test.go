package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNamespaceKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewNamespaceKindValidator,
		ErrorCode: vet.IllegalKindInNamespacesErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "Namespace allowed",
				Object: fake.Namespace("namespaces/foo"),
			},
			{
				Name:       "ConfigManagement not allowed",
				Object:     fake.ConfigManagement("namespaces/foo/config-management.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
