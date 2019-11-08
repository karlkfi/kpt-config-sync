package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNamespaceValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("Cluster scoped object",
			fake.ClusterRole(),
		),
		nht.Pass("Namespaced object in valid namespace",
			fake.Role(core.Namespace("backend")),
		),
		nht.Fail("Namespaced object in invalid namespace",
			fake.Role(core.Namespace(configmanagement.ControllerNamespace)),
		),
		nht.Pass("valid Namespace",
			fake.Namespace("backend"),
		),
		nht.Fail("invalid Namespace",
			fake.Namespace(configmanagement.ControllerNamespace),
		),
	}

	nht.RunAll(t, nonhierarchical.IllegalNamespaceValidator, testCases)
}
