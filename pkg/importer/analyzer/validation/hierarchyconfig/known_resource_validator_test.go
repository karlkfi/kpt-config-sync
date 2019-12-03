package hierarchyconfig

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	rbac "k8s.io/api/rbac/v1alpha1"
)

func hierarchyConfig(hcrs ...v1.HierarchyConfigResource) ast.FileObject {
	hc := fake.HierarchyConfigObject()
	hc.Spec.Resources = hcrs

	return fake.FileObject(hc, "system/hc.yaml")
}

func resource(group string, kinds ...string) v1.HierarchyConfigResource {
	return v1.HierarchyConfigResource{
		Group: group,
		Kinds: kinds,
	}
}

func TestHierarchyConfigScopeValidator(t *testing.T) {
	scoper := discovery.Scoper{
		kinds.Role().GroupKind():        discovery.NamespaceScope,
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,
	}

	testCases := []nht.ValidatorTestCase{
		nht.Pass("no resources", hierarchyConfig()),
		nht.Pass("empty HierarchyConfig", hierarchyConfig()),
		nht.Pass("namespace-scoped resource",
			hierarchyConfig(resource(rbac.GroupName, kinds.Role().Kind)),
		),
		nht.Fail("cluster-scoped resource",
			hierarchyConfig(resource(rbac.GroupName, kinds.ClusterRole().Kind)),
		),
		nht.Fail("cluster and namespace-scoped resource",
			hierarchyConfig(resource(rbac.GroupName, kinds.ClusterRole().Kind, kinds.Role().Kind)),
		),
		nht.Fail("unknown resource",
			hierarchyConfig(resource("unknown", "UnknownType")),
		),
	}

	nht.RunAll(t, NewHierarchyConfigScopeValidator(scoper), testCases)
}
