package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestClusterregistryKindValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewClusterRegistryKindValidator,
		ErrorCode: vet.IllegalKindInClusterregistryErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "ClusterSelector allowed",
				Object: fake.ClusterSelector("clusterregistry/cs.yaml"),
			},
			{
				Name:   "Cluster allowed",
				Object: fake.Cluster("clusterregistry/cluster.yaml"),
			},
			{
				Name:       "Role not allowed",
				Object:     fake.Role("clusterregistry/role.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
