package validation_test

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
)

func TestScope(t *testing.T) {
	scoper := discovery.Scoper{
		kinds.Role().GroupKind():        discovery.NamespaceScope,
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,
	}

	testCases := []nht.ValidatorTestCase{
		nht.Pass("Role in namespaces/",
			fake.RoleAtPath("namespaces/role.yaml")),
		nht.Fail("Role in cluster/",
			fake.RoleAtPath("cluster/role.yaml")),
		nht.Fail("ClusterRole in namespaces/",
			fake.ClusterRoleAtPath("namespaces/clusterrole.yaml")),
		nht.Pass("ClusterRole in cluster/",
			fake.ClusterRoleAtPath("cluster/clusterrole.yaml")),
		nht.Pass("Namespace in namespaces/",
			fake.NamespaceAtPath("namespaces/namespace.yaml")),
		nht.Fail("Namespace in cluster/",
			fake.NamespaceAtPath("cluster/namespace.yaml")),
		nht.Fail("unknown object in namespaces/",
			fake.AnvilAtPath("namespaces/anvil.yaml")),
		nht.Fail("unknown in cluster/",
			fake.AnvilAtPath("cluster/anvil.yaml")),
		nht.Pass("generated ResourceQuota",
			fake.FileObject(
				fake.ResourceQuotaObject(core.Name(resourcequota.ResourceQuotaObjectName)), ""),
		),
	}

	nht.RunAll(t, validation.NewTopLevelDirectoryValidator(scoper), testCases)
}
