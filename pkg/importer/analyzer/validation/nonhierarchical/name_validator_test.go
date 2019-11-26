package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNameValidation(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("with name", fake.Role(core.Name("foo"))),
		nht.Fail("with empty name", fake.Role(core.Name(""))),
		nht.Fail("with invalid name", fake.Role(core.Name("Foo"))),
	}

	nht.RunAll(t, nonhierarchical.NameValidator, testCases)
}
