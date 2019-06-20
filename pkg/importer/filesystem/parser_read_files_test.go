package filesystem_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// Tests that don't make sense without literally writing to a hard disk.
// Or, ones that (for now) would require their own CL just to refactor to not require writing to a
// hard disk.

func newTestDir(t *testing.T) *testDir {
	root, err := ioutil.TempDir("", "test_dir")
	if err != nil {
		t.Fatalf("Failed to create test dir %v", err)
	}
	return &testDir{root}
}

func TestEmptyDirectories(t *testing.T) {
	// Parsing should not encounter errors on seeing empty directories. If an error should occur, it
	// should be later.
	d := newTestDir(t)
	defer d.remove()

	for _, path := range []string{
		filepath.Join(d.rootDir, repo.SystemDir),
		filepath.Join(d.rootDir, repo.ClusterDir),
		filepath.Join(d.rootDir, repo.ClusterRegistryDir),
		filepath.Join(d.rootDir, repo.NamespacesDir),
	} {
		t.Run(path, func(t *testing.T) {
			if err := os.MkdirAll(path, 0750); err != nil {
				t.Fatalf("error creating test dir %s: %v", path, err)
			}

			f := fstesting.NewTestClientGetter(t)
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			var err error
			rootPath, err := cmpath.NewRoot(cmpath.FromOS(d.rootDir))
			if err != nil {
				t.Error(err)
			}

			p := filesystem.NewParser(
				f,
				filesystem.ParserOpt{
					Vet:       false,
					Validate:  true,
					Extension: &filesystem.NomosVisitorProvider{},
					RootPath:  rootPath,
				},
			)

			if p.Errors() != nil {
				t.Fatalf("unexpected error: %v", p.Errors())
			}
		})
	}
}

// TestParserPerClusterAddressingVet tests nomos vet validation errors.
func TestFailOnInvalidYAML(t *testing.T) {
	tests := []parserTestCase{
		{
			testName:    "Defining invalid yaml is an error.",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				"namespaces/invalid.yaml": "This is not valid yaml.",
			},
			expectedErrorCodes: []string{status.APIServerErrorCode},
		},
	}
	for _, test := range tests {
		test.testFiles["system/repo.yaml"] = aRepo
		t.Run(test.testName, test.Run)
	}
}
