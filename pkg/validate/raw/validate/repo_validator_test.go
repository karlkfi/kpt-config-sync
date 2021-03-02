package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/repo"
	"github.com/google/nomos/pkg/validate/objects"
)

const notAllowedRepoVersion = "0.0.0"

func TestRepo(t *testing.T) {
	testCases := []struct {
		name    string
		objs    *objects.Raw
		wantErr status.Error
	}{
		{
			name: "Repo with current version",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Repo(fake.RepoVersion(repo.CurrentVersion)),
				},
			},
		},
		{
			name: "Repo with supported old version",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Repo(fake.RepoVersion(system.OldAllowedRepoVersion)),
				},
			},
		},
		{
			name: "Repo with unsupported old version",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Repo(fake.RepoVersion(notAllowedRepoVersion)),
				},
			},
			wantErr: system.UnsupportedRepoSpecVersion(fake.Repo(fake.RepoVersion(notAllowedRepoVersion)), notAllowedRepoVersion),
		},
		{
			name: "Missing Repo",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Role(),
					fake.RoleBinding(),
				},
			},
			wantErr: system.MissingRepoError(),
		},
		{
			name: "Multiple Repos",
			objs: &objects.Raw{
				Objects: []ast.FileObject{
					fake.Repo(core.Name("first")),
					fake.Repo(core.Name("second")),
				},
			},
			wantErr: status.MultipleSingletonsError(fake.Repo(), fake.Repo()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Repo(tc.objs)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got Repo() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
