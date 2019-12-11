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
		nht.Pass("with name", fake.ResourceQuota(core.Name("foo"))),
		nht.Fail("with empty name", fake.ResourceQuota(core.Name(""))),
		nht.Fail("with invalid name", fake.ResourceQuota(core.Name("Foo"))),
		// rbac types have hard-coded different rules than all other Kubernetes types.
		nht.Pass("rbac with capital letters and colon", fake.Role(core.Name("A:B"))),
		nht.Fail("rbac with forward slash", fake.Role(core.Name("a/b"))),
	}

	nht.RunAll(t, nonhierarchical.NameValidator, testCases)
}
