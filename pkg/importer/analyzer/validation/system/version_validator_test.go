package system_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/repo"
)

const notAllowedRepoVersion = "0.0.0"

func TestRepoVersionValidator(t *testing.T) {
	test := asttest.Validator(system.NewRepoVersionValidator,
		system.UnsupportedRepoSpecVersionCode,
		asttest.Pass("Repo with current version",
			fake.Repo(fake.RepoVersion(repo.CurrentVersion))),
		asttest.Pass("Repo with supported old version",
			fake.Repo(fake.RepoVersion(system.OldAllowedRepoVersion))),
		asttest.Fail("Repo with current version",
			fake.Repo(fake.RepoVersion(notAllowedRepoVersion))))

	test.RunAll(t)
}
