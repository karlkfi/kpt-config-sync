package initialize

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/cmd/nomos/repo"
	fstesting "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	"github.com/pkg/errors"
)

type testCase struct {
	name        string
	root        string
	before      func(*fstesting.TestDir)
	expectError bool
}

func TestInitialization(t *testing.T) {
	testCases := []testCase{
		{
			name:   "empty dir",
			root:   "empty",
			before: func(dir *fstesting.TestDir) {},
		},
		{
			name: "dir with file",
			root: "nonempty",
			before: func(dir *fstesting.TestDir) {
				dir.CreateTestFile("somefile.txt", "contents")
			},
			expectError: true,
		},
		{
			name: "dir with subdir",
			root: "nonempty",
			before: func(dir *fstesting.TestDir) {
				dir.CreateTestFile("somedir/somefile.txt", "contents")
			},
			expectError: true,
		},
		{
			name: "nonexistent dir",
			root: "nonexistent",
			before: func(dir *fstesting.TestDir) {
				dir.Remove()
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func runTestCase(t *testing.T, tc testCase) {
	testDir := fstesting.NewTestDir(t, tc.root)
	defer testDir.Remove()

	tc.before(testDir)

	// Cleanup test dir before starting.
	os.Remove(filepath.Join(testDir.Root(), "tree"))

	dir := repo.FilePath{}
	dir.Set(testDir.Root())

	// Run Initialize
	err := Initialize(dir)
	if err != nil {
		if !tc.expectError {
			t.Error(errors.Wrapf(err, "(Test: %s): did not expect error but got", tc.name))
		}
		// Got and expected error
	} else if tc.expectError {
		t.Errorf("(Test: %s): expected error but got none", tc.name)
	}
}
