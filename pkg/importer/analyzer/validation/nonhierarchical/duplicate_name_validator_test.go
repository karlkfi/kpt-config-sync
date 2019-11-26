package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDuplicateNameValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("two objects with different names",
			fake.Role(core.Name("alice"), core.Namespace("shipping")),
			fake.Role(core.Name("bob"), core.Namespace("shipping")),
		),
		nht.Pass("two objects with different namespaces",
			fake.Role(core.Name("alice"), core.Namespace("shipping")),
			fake.Role(core.Name("alice"), core.Namespace("production")),
		),
		nht.Pass("two objects with different kinds",
			fake.Role(core.Name("alice"), core.Namespace("shipping")),
			fake.RoleBinding(core.Name("alice"), core.Namespace("shipping")),
		),
		nht.Fail("two duplicate objects",
			fake.Role(core.Name("alice"), core.Namespace("shipping")),
			fake.Role(core.Name("alice"), core.Namespace("shipping")),
		),
		nht.Fail("duplicate cluster-scoped objects",
			fake.ClusterRole(core.Name("alice")),
			fake.ClusterRole(core.Name("alice")),
		),
		nht.Pass("cluster-scoped objects with different names",
			fake.ClusterRole(core.Name("alice")),
			fake.ClusterRole(core.Name("bob")),
		),
	}

	nht.RunAll(t, nonhierarchical.DuplicateNameValidator, testCases)
}
