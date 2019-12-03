package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
)

func TestScopeValidator(t *testing.T) {
	scoper := discovery.Scoper{
		kinds.Role().GroupKind():        discovery.NamespaceScope,
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,
	}
	testCases := []nht.ValidatorTestCase{
		nht.Pass("Namespace-scoped object with metadata.namespace",
			fake.Role(core.Namespace("backend")),
		),
		nht.Fail("Namespace-scoped object without metadata.namespace",
			fake.Role(core.Namespace("")),
		),
		nht.Fail("Cluster-scoped object with metadata.namespace",
			fake.ClusterRole(core.Namespace("backend")),
		),
		nht.Pass("Cluster-scoped object with metadata.namespace",
			fake.ClusterRole(core.Namespace("")),
		),
		nht.Fail("Unknown type with metadata.namespace",
			fake.NamespaceSelector(core.Namespace("backend")),
		),
		nht.Fail("Unknown type without metadata.namespace",
			fake.NamespaceSelector(core.Namespace("")),
		),
	}
	nht.RunAll(t, nonhierarchical.ScopeValidator(scoper), testCases)
}
