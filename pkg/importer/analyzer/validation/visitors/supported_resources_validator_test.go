package visitors

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/testing/fake"

	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
)

func TestSupportedClusterResourcesValidator(t *testing.T) {
	newSupportedClusterResourcesValidator := func() *visitor.ValidatorVisitor {
		return NewSupportedClusterResourcesValidator()
	}
	test := visitortesting.ObjectValidatorTest{
		Validator: newSupportedClusterResourcesValidator,
		ErrorCode: nonhierarchical.UnsupportedObjectErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:   "clusterrole Object",
				Object: fake.UnstructuredAtPath(kinds.ClusterRole(), "cluster/r.yaml"),
			},
			{
				Name:       "sync Object",
				Object:     fake.UnstructuredAtPath(kinds.Sync(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "Repo Object",
				Object:     fake.UnstructuredAtPath(kinds.Repo(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "NamespaceConfig Object",
				Object:     fake.UnstructuredAtPath(kinds.NamespaceConfig(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ClusterConfig Object",
				Object:     fake.UnstructuredAtPath(kinds.ClusterConfig(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "HierarchyConfig Object",
				Object:     fake.UnstructuredAtPath(kinds.HierarchyConfig(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "Namespace Object",
				Object:     fake.UnstructuredAtPath(kinds.Namespace(), "cluster/r.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
