package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestClusterregistryKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewClusterRegistryKindValidator,
		ErrorCode: vet.IllegalKindInClusterregistryErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "ClusterSelector allowed",
				Object: fake.ClusterSelectorAtPath("clusterregistry/cs.yaml"),
			},
			{
				Name:   "Cluster allowed",
				Object: fake.ClusterAtPath("clusterregistry/cluster.yaml"),
			},
			{
				Name:       "Role not allowed",
				Object:     fake.RoleAtPath("clusterregistry/role.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
