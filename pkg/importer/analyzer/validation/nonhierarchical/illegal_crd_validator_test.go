package nonhierarchical_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestIllegalCrdValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		// v1beta1.CRD
		nht.Fail("ClusterConfig", crd("crd", kinds.ClusterConfig())),
		nht.Fail("Namespaceconfig", crd("crd", kinds.NamespaceConfig())),
		nht.Fail("Sync", crd("crd", kinds.Sync())),
		nht.Pass("Anvil", crd("crd", kinds.Anvil())),
		nht.Pass("non-crd", fake.ClusterRole()),
		// v1.CRD
		nht.Fail("ClusterConfig",
			fake.ToCustomResourceDefinitionV1(crd("crd", kinds.ClusterConfig()))),
		nht.Fail("Namespaceconfig",
			fake.ToCustomResourceDefinitionV1(crd("crd", kinds.NamespaceConfig()))),
		nht.Fail("Sync",
			fake.ToCustomResourceDefinitionV1(crd("crd", kinds.Sync()))),
		nht.Pass("Anvil",
			fake.ToCustomResourceDefinitionV1(crd("crd", kinds.Anvil()))),
		nht.Pass("non-crd",
			fake.ClusterRole()),
	}

	nht.RunAll(t, nonhierarchical.IllegalCRDValidator, testCases)
}
