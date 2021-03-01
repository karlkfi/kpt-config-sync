package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/repo"
)

const notAllowedRepoVersion = "0.0.0"

func TestRepoVersion(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Repo with current version",
			obj:  fake.Repo(fake.RepoVersion(repo.CurrentVersion)),
		},
		{
			name: "Repo with supported old version",
			obj:  fake.Repo(fake.RepoVersion(system.OldAllowedRepoVersion)),
		},
		{
			name:    "Repo with unsupported old version",
			obj:     fake.Repo(fake.RepoVersion(notAllowedRepoVersion)),
			wantErr: system.UnsupportedRepoSpecVersion(fake.Repo(fake.RepoVersion(notAllowedRepoVersion)), notAllowedRepoVersion),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := RepoVersion(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got RepoVersion() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
