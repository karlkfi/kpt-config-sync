package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestHierarchicalKindValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("non-configmanagement type",
			fake.Role(),
		),
		nht.Fail("configmanagement type",
			fake.Repo(),
		),
	}

	nht.RunAll(t, nonhierarchical.IllegalHierarchicalKindValidator, testCases)
}
