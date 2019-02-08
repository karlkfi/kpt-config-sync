package system

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	testing2 "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

const notAllowedRepoVersion = "0.0.0"

func repo(version string) ast.FileObject {
	result := fake.Repo("system/repo.yaml")
	result.Object.(*v1alpha1.Repo).Spec.Version = version
	return result
}

func TestRepoVersionValidator(t *testing.T) {
	test := testing2.ObjectValidatorTest{
		Validator: NewRepoVersionValidator,
		ErrorCode: vet.UnsupportedRepoSpecVersionCode,
		TestCases: []testing2.ObjectValidatorTestCase{
			{
				Name:   "Hierarhcy Config is fine",
				Object: fake.HierarchyConfig("system/hc.yaml"),
			},
			{
				Name:   "Repo with valid version is fine",
				Object: repo(allowedRepoVersion),
			},
			{
				Name:       "Repo with valid version is fine",
				Object:     repo(notAllowedRepoVersion),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
