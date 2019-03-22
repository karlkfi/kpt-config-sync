package system_test

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/system"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	testing2 "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
)

const notAllowedRepoVersion = "0.0.0"

func version(version string) object.Mutator {
	return func(o *ast.FileObject) {
		o.Object.(*v1.Repo).Spec.Version = version
	}
}

func TestRepoVersionValidator(t *testing.T) {
	test := testing2.ObjectValidatorTest{
		Validator: system.NewRepoVersionValidator,
		ErrorCode: vet.UnsupportedRepoSpecVersionCode,
		TestCases: []testing2.ObjectValidatorTestCase{
			{
				Name:   "Hierarhcy Config is fine",
				Object: fake.HierarchyConfig("system/hc.yaml"),
			},
			{
				Name:   "Repo with valid version is fine",
				Object: fake.Build(kinds.Repo(), version(system.AllowedRepoVersion)),
			},
			{
				Name:       "Repo with invalid version is error",
				Object:     fake.Build(kinds.Repo(), version(notAllowedRepoVersion)),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
