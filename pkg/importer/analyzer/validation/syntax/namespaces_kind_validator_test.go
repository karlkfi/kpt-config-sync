package syntax

import (
	"testing"

	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNamespaceKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewNamespaceKindValidator,
		ErrorCode: IllegalKindInNamespacesErrorCode,
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
