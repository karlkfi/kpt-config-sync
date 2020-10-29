package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestKptfileExistenceValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("Kptfile does not exist",
			fake.ClusterRole(),
			fake.Namespace("backend"),
		),
		nht.Fail("Kptfile exists",
			fake.KptFile("Kptfile", core.Name("n1")),
			fake.KptFile("Kptfile"),
		),
	}
	nht.RunAll(t, nonhierarchical.KptfileExistenceValidator, testCases)
}
