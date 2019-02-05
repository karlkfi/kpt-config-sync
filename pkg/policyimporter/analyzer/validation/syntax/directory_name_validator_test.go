package syntax

import (
	"testing"

	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDirectoryNameValidator_Pass(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewDirectoryNameValidator,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "foo",
				Object: fake.Role("namespaces/foo/role.yaml"),
			},
			{
				Name:   "foo1",
				Object: fake.Role("namespaces/foo1/role.yaml"),
			},
		},
	}

	test.RunAll(t)
}
