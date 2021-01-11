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
		nht.Fail("ClusterConfig v1beta1", crdv1beta1("crd", kinds.ClusterConfig())),
		nht.Fail("NamespaceConfig v1beta1", crdv1beta1("crd", kinds.NamespaceConfig())),
		nht.Fail("Sync", crdv1beta1("crd", kinds.Sync())),
		nht.Pass("Anvil v1beta1", crdv1beta1("crd", kinds.Anvil())),
		nht.Pass("non-crd", fake.ClusterRole()),
		// v1.CRD
		nht.Fail("ClusterConfig v1", crdv1("crd", kinds.ClusterConfig())),
		nht.Fail("NamespaceConfig v1", crdv1("crd", kinds.NamespaceConfig())),
		nht.Fail("Sync v1", crdv1("crd", kinds.Sync())),
		nht.Pass("Anvil v1", crdv1("crd", kinds.Anvil())),
		nht.Pass("non-crd", fake.ClusterRole()),
	}

	nht.RunAll(t, nonhierarchical.IllegalCRDValidator, testCases)
}
