package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDuplicateNameValidator(t *testing.T) {
	// Primary logic tested in pkg/importer/analyzer/validation/metadata/duplicate_name_validator_test.go.
	// No need to fully duplicate here.
	testCases := []nht.ValidatorTestCase{
		nht.Pass("two non-duplicate objects",
			fake.Role(object.Name("alice")),
			fake.Role(object.Name("bob")),
		),
		nht.Fail("two duplicate objects",
			fake.Role(object.Name("alice")),
			fake.Role(object.Name("alice")),
		),
	}

	nht.RunAll(t, nonhierarchical.DuplicateNameValidator, testCases)
}
