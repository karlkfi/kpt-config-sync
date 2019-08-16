package syntax

import (
	"testing"

	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
)

func TestDisallowedCRDNameValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewCRDNameValidator,
		ErrorCode: InvalidCRDNameErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:       "non plural name",
				Object:     crd("cluster/anvil-crd.yaml", "anvil.acme.com", kinds.Anvil()),
				ShouldFail: true,
			},
			{
				Name:       "missing group name",
				Object:     crd("cluster/anvil-crd.yaml", "anvils", kinds.Anvil()),
				ShouldFail: true,
			},
			{
				Name:   "valid name",
				Object: crd("cluster/anvil-crd.yaml", "anvils.acme.com", kinds.Anvil()),
			},
		},
	}

	test.RunAll(t)
}
