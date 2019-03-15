package visitors

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

func TestSupportedClusterResourcesValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewSupportedClusterResourcesValidator,
		ErrorCode: vet.UnsupportedObjectErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:   "clusterrole Object",
				Object: asttesting.NewFakeFileObject(kinds.ClusterRole(), "cluster/r.yaml"),
			},
			{
				Name:       "sync Object",
				Object:     asttesting.NewFakeFileObject(kinds.Sync(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "Repo Object",
				Object:     asttesting.NewFakeFileObject(kinds.Repo(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "NamespaceConfig Object",
				Object:     asttesting.NewFakeFileObject(kinds.NamespaceConfig(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "ClusterConfig Object",
				Object:     asttesting.NewFakeFileObject(kinds.ClusterConfig(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "HierarchyConfig Object",
				Object:     asttesting.NewFakeFileObject(kinds.HierarchyConfig(), "cluster/r.yaml"),
				ShouldFail: true,
			},
			{
				Name:       "Namespace Object",
				Object:     asttesting.NewFakeFileObject(kinds.Namespace(), "cluster/r.yaml"),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
