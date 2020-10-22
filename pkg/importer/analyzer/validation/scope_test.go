package validation_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
)

func TestScope(t *testing.T) {
	scoper := discovery.CoreScoper(true)

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
	}

	nht.RunAll(t, validation.NewTopLevelDirectoryValidator(scoper, cmpath.RelativeSlash("")), testCases)
}

func TestScopeServerless(t *testing.T) {
	scoper := discovery.CoreScoper(false)

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
		nht.Pass("unknown object in namespaces/",
			fake.AnvilAtPath("namespaces/anvil.yaml")),
		nht.Pass("unknown in cluster/",
			fake.AnvilAtPath("cluster/anvil.yaml")),
	}

	nht.RunAll(t, validation.NewTopLevelDirectoryValidator(scoper, cmpath.RelativeSlash("")), testCases)
}

func TestScopePolicyDir(t *testing.T) {
	scoper := discovery.CoreScoper(true)

	testCases := []nht.ValidatorTestCase{
		nht.Pass("Role in acme/namespaces/",
			fake.RoleAtPath("acme/namespaces/role.yaml")),
		nht.Fail("Role in foo/namespaces/",
			fake.RoleAtPath("foo/namespaces/role.yaml")),
		nht.Pass("ClusterRole in acme/cluster/",
			fake.ClusterRoleAtPath("acme/cluster/clusterrole.yaml")),
		nht.Fail("ClusterRole in foo/cluster/",
			fake.ClusterRoleAtPath("foo/cluster/clusterrole.yaml")),
		nht.Pass("Namespace in acme/namespaces/",
			fake.NamespaceAtPath("acme/namespaces/namespace.yaml")),
		nht.Fail("Namespace in foo/namespaces/",
			fake.NamespaceAtPath("foo/namespaces/namespace.yaml")),
		nht.Fail("unknown object in acme/namespaces/",
			fake.AnvilAtPath("acme/namespaces/anvil.yaml")),
		nht.Fail("unknown object in foo/namespaces/",
			fake.AnvilAtPath("foo/namespaces/anvil.yaml")),
		nht.Fail("unknown in acme/cluster/",
			fake.AnvilAtPath("acme/cluster/anvil.yaml")),
		nht.Fail("unknown in foo/cluster/",
			fake.AnvilAtPath("foo/cluster/anvil.yaml")),
	}

	nht.RunAll(t, validation.NewTopLevelDirectoryValidator(scoper, cmpath.RelativeSlash("acme")), testCases)
}
