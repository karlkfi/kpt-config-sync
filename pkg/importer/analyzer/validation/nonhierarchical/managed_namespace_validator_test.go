package nonhierarchical_test

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestManagedNamespaceValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("empty"),
		nht.Pass("managed cluster-scoped object",
			fake.ClusterRole(),
		),
		nht.Pass("unmanaged cluster-scoped object",
			fake.ClusterRole(core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
		),
		nht.Pass("managed resource in managed Namepsace",
			fake.Namespace("namespaces/foo"),
			fake.Role(core.Namespace("foo")),
		),
		nht.Pass("unmanaged resource in managed Namepsace",
			fake.Namespace("namespaces/foo"),
			fake.Role(core.Namespace("foo"), core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
		),
		nht.Fail("managed resource in unmanaged Namepsace",
			fake.Namespace("namespaces/foo", core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
			fake.Role(core.Namespace("foo")),
		),
		nht.Pass("unmanaged resource in unmanaged Namepsace",
			fake.Namespace("namespaces/foo", core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
			fake.Role(core.Namespace("foo"), core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled)),
		),
	}

	nht.RunAll(t, nonhierarchical.ManagedNamespaceValidator, testCases)
}
