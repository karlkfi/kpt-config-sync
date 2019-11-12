package system_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	testing2 "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/repo"
)

const notAllowedRepoVersion = "0.0.0"

func TestRepoVersionValidator(t *testing.T) {
	test := testing2.ObjectValidatorTest{
		Validator: system.NewRepoVersionValidator,
		ErrorCode: system.UnsupportedRepoSpecVersionCode,
		TestCases: []testing2.ObjectValidatorTestCase{
			{
				Name:   "Hierarhcy Config is fine",
				Object: fake.HierarchyConfig(),
			},
			{
				Name:   "Repo with current version is fine",
				Object: fake.Repo(fake.RepoVersion(repo.CurrentVersion)),
			},
			{
				Name:   "Repo with old version is fine",
				Object: fake.Repo(fake.RepoVersion(system.OldAllowedRepoVersion)),
			},
			{
				Name:       "Repo with invalid version is error",
				Object:     fake.Repo(fake.RepoVersion(notAllowedRepoVersion)),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
