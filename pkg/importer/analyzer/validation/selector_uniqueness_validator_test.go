package validation

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestClusterSelectorUniqueness(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("empty"),
		nht.Pass("single ClusterSelector",
			fake.ClusterSelector(core.Name("sre-supported")),
		),
		nht.Fail("duplicate ClusterSelectors",
			fake.ClusterSelector(core.Name("sre-supported")),
			fake.ClusterSelector(core.Name("sre-supported")),
		),
		nht.Pass("different ClusterSelectors",
			fake.ClusterSelector(core.Name("sre-supported")),
			fake.ClusterSelector(core.Name("prod-environment")),
		),
		// NamespaceSelectors may be Cluster-selected, but since this is before
		// Cluster-selection, they would appear as duplicates.
		nht.Pass("duplicate NamespaceSelector",
			fake.NamespaceSelector(core.Name("sre-supported")),
			fake.NamespaceSelector(core.Name("sre-supported")),
		),
	}

	nht.RunAll(t, ClusterSelectorUniqueness, testCases)
}

func TestNamespaceSelectorUniqueness(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("empty"),
		nht.Pass("single NamespaceSelector",
			fake.NamespaceSelector(core.Name("sre-supported")),
		),
		nht.Fail("duplicate NamespaceSelector",
			fake.NamespaceSelector(core.Name("sre-supported")),
			fake.NamespaceSelector(core.Name("sre-supported")),
		),
		nht.Pass("different NamespaceSelectors",
			fake.NamespaceSelector(core.Name("sre-supported")),
			fake.NamespaceSelector(core.Name("prod-environment")),
		),
	}

	nht.RunAll(t, NamespaceSelectorUniqueness, testCases)
}
