package system_test

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/system"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	testing2 "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/testing/object"
)

const notAllowedRepoVersion = "0.0.0"

func version(version string) object.BuildOpt {
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
				Object: object.Build(kinds.Repo(), version(system.AllowedRepoVersion)),
			},
			{
				Name:       "Repo with invalid version is error",
				Object:     object.Build(kinds.Repo(), version(notAllowedRepoVersion)),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
