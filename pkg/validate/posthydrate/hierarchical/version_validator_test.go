package hierarchical

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/google/nomos/pkg/validate/parsed"
)

const notAllowedRepoVersion = "0.0.0"

func TestRepoVersionValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "Repo with current version",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(fake.RepoVersion(repo.CurrentVersion)),
				},
			},
		},
		{
			name: "Repo with supported old version",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(fake.RepoVersion(system.OldAllowedRepoVersion)),
				},
			},
		},
		{
			name: "Repo with unsupported old version",
			root: &parsed.TreeRoot{
				SystemObjects: []ast.FileObject{
					fake.Repo(fake.RepoVersion(notAllowedRepoVersion)),
				},
			},
			wantErr: status.Append(nil, system.UnsupportedRepoSpecVersion(fake.Repo(fake.RepoVersion(notAllowedRepoVersion)), notAllowedRepoVersion)),
		},
	}

	for _, tc := range testCases {
		rv := RepoVersionValidator()
		t.Run(tc.name, func(t *testing.T) {
			err := rv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got RepoVersionValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
