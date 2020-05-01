package parse

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/pkg/errors"
)

func TestListPolicyFiles(t *testing.T) {
	testCases := []struct {
		name  string
		files []string
		want  []string
	}{
		{
			name: "empty returns empty",
		},
		{
			name:  "read .yml, .yaml, and .json",
			files: []string{"ns.yaml", "role.yml", "rb.json"},
			want:  []string{"ns.yaml", "role.yml", "rb.json"},
		},
		{
			name:  "read subdirectory",
			files: []string{"namespaces/foo/", "namespaces/foo/ns.yaml"},
			want:  []string{"namespaces/foo/ns.yaml"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			dir, err := ioutil.TempDir(os.TempDir(), "nomos-git-test")
			if err != nil {
				t.Fatal(err)
			}

			// Initialize a git repository.
			out, err := exec.Command("git", "-C", dir, "init").CombinedOutput()
			if err != nil {
				t.Fatal(errors.Wrap(err, string(out)))
			}

			// Add all of the specified files to the repository.
			for _, f := range tc.files {
				var err error
				if strings.HasSuffix(f, "/") {
					err = os.MkdirAll(filepath.Join(dir, cmpath.FromSlash(f).OSPath()), os.ModePerm)
				} else {
					file, err := os.Create(filepath.Join(dir, cmpath.FromSlash(f).OSPath()))
					if err != nil {
						t.Fatal(err)
					}
					err = file.Close()
					if err != nil {
						t.Fatal(err)
					}
					out, err = exec.Command("git", "-C", dir, "add", f).CombinedOutput()
					if err != nil {
						t.Fatal(errors.Wrap(err, string(out)))
					}
				}
				if err != nil {
					t.Fatal(err)
				}
			}

			// Commit. Note that the identification fields are required as this
			// may be running in a container without a git config set up.
			if len(tc.files) > 0 {
				out, err = exec.Command("git",
					"-c", "user.name='Nomos Test'",
					"-c", "user.email='nomos-team@google.comcmpath'",
					"-C", dir, "commit", "-m", "add files").CombinedOutput()
				if err != nil {
					t.Fatal(errors.Wrap(err, string(out)))
				}
			}

			abs, err := cmpath.Abs(cmpath.FromOS(dir))
			if err != nil {
				t.Fatal(err)
			}

			resultGit, err := git.ListFiles(abs)
			if err != nil {
				t.Fatal(err)
			}
			sort.Slice(resultGit, func(i, j int) bool {
				return resultGit[i].OSPath() < resultGit[j].OSPath()
			})
			resultFind, err := FindFiles(abs)
			if err != nil {
				t.Fatal(err)
			}
			sort.Slice(resultFind, func(i, j int) bool {
				return resultFind[i].OSPath() < resultFind[j].OSPath()
			})
			if diff := cmp.Diff(resultGit, resultFind); diff != "" {
				t.Errorf("diff between ListFiles and FindFiles:\n%s", diff)
			}

			sort.Strings(tc.want)
			var want []string
			for _, w := range tc.want {
				// Since ListFiles returns absolute paths, we have to convert
				// these to the expected absolute paths that include the randomly-generated
				// temp diretory.
				want = append(want, filepath.Join(dir, cmpath.FromSlash(w).OSPath()))
			}

			var got []string
			for _, r := range resultGit {
				got = append(got, r.SlashPath())
			}
			sort.Strings(got)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}
